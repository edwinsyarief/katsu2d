package katsu2d

import (
	_ "embed"
	"image/color"
	"math"
	"sync"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed internal_assets/shaders/shadow_blur.kage
var shadowBlurShaderSrc []byte

var (
	shadowBlurOnce   sync.Once
	shadowBlurShader *ebiten.Shader

	contactShadowOnce sync.Once
	contactShadowImg  *ebiten.Image
)

const (
	contactShadowSize       = 64
	defaultShadowOpacity    = 0.45
	defaultShadowCullMargin = 128.0
	defaultBlurDownsample   = 0.5
	minBlurDownsample       = 0.25
	maxBlurDownsample       = 1.0
	maxContactBatchVerts    = 16380 * 4
)

func getShadowBlurShader() *ebiten.Shader {
	shadowBlurOnce.Do(func() {
		var err error
		shadowBlurShader, err = ebiten.NewShader(shadowBlurShaderSrc)
		if err != nil {
			panic("katsu2d: failed to compile shadow_blur.kage: " + err.Error())
		}
	})
	return shadowBlurShader
}

// contactShadowTexture lazily generates and returns a shared 64x64 soft radial
// contact-shadow texture with premultiplied white (a, a, a, a) quadratic falloff.
func contactShadowTexture() *ebiten.Image {
	contactShadowOnce.Do(func() {
		pixels := make([]byte, contactShadowSize*contactShadowSize*4)
		const half = float64(contactShadowSize) * 0.5
		const invHalf = 1.0 / half
		for y := range contactShadowSize {
			dy := (float64(y) + 0.5 - half) * invHalf
			dy2 := dy * dy
			row := y * contactShadowSize * 4
			for x := range contactShadowSize {
				dx := (float64(x) + 0.5 - half) * invHalf
				r2 := dx*dx + dy2
				if r2 >= 1.0 {
					continue
				}
				t := 1.0 - r2
				v := uint8(math.Round(t * t * 255.0))
				idx := row + x*4
				pixels[idx] = v
				pixels[idx+1] = v
				pixels[idx+2] = v
				pixels[idx+3] = v
			}
		}
		img := ebiten.NewImage(contactShadowSize, contactShadowSize)
		img.WritePixels(pixels)
		contactShadowImg = img
	})
	return contactShadowImg
}

// ShadowOption configures a ShadowRenderSystem.
type ShadowOption func(*ShadowRenderSystem)

// ShadowSystemOption is an alias for ShadowOption.
type ShadowSystemOption = ShadowOption

// WithShadowComposited enables or disables single-pass offscreen mask compositing
// to prevent overlapping shadows from double-darkening.
func WithShadowComposited(enabled bool) ShadowOption {
	return func(s *ShadowRenderSystem) {
		s.composited = enabled
	}
}

// WithShadowBlur configures the 2-pass separable Kage Gaussian blur radius in pixels.
// Setting a positive radius implicitly enables composited mode.
func WithShadowBlur(radius float64) ShadowOption {
	return func(s *ShadowRenderSystem) {
		if radius < 0 {
			radius = 0
		}
		s.blurRadius = radius
		if radius > 0 {
			s.composited = true
		}
	}
}

// WithShadowBlurDownsample sets the resolution scale factor for offscreen shadow blur
// passes, clamped to [0.25, 1.0] (default 0.5 when blur is enabled).
func WithShadowBlurDownsample(scale float64) ShadowOption {
	return func(s *ShadowRenderSystem) {
		s.blurDownsample = Clamp(scale, minBlurDownsample, maxBlurDownsample)
	}
}

// WithShadowGlobalOpacity sets the uniform opacity applied when compositing
// the offscreen shadow mask onto the screen (default 0.45).
func WithShadowGlobalOpacity(opacity float64) ShadowOption {
	return func(s *ShadowRenderSystem) {
		s.globalOpacity = Clamp(opacity, 0.0, 1.0)
	}
}

// WithShadowGlobalColor sets an optional tint multiplier applied when compositing
// the offscreen shadow mask onto the screen.
func WithShadowGlobalColor(col color.RGBA) ShadowOption {
	return func(s *ShadowRenderSystem) {
		s.globalColor = col
		s.hasGlobalColor = true
	}
}

// WithShadowFrustumMargin sets the extra world-space padding around ActiveCamera.ViewBounds
// used for shadow frustum culling (default 128).
func WithShadowFrustumMargin(margin float64) ShadowOption {
	return func(s *ShadowRenderSystem) {
		if margin < 0 {
			margin = 0
		}
		s.frustumMargin = margin
	}
}

// ShadowRenderSystem renders projected, drop, contact, and reflection shadows
// with optional offscreen compositing and 2-pass separable Gaussian blur.
type ShadowRenderSystem struct {
	shadowFilter  *teishoku.Filter2[TransformComponent, ShadowComponent]
	lightFilter   *teishoku.Filter[ShadowLightComponent]
	spriteBuilder *teishoku.Builder[SpriteComponent]
	tmHandle      teishoku.ResourceHandle[TextureManager]
	camHandle     teishoku.ResourceHandle[ActiveCamera]

	composited     bool
	blurRadius     float64
	blurDownsample float64
	globalOpacity  float64
	globalColor    color.RGBA
	hasGlobalColor bool
	frustumMargin  float64

	quadVerts          [4]ebiten.Vertex
	quadIndices        [6]uint16
	screenVerts        [4]ebiten.Vertex
	contactMaskVerts   []ebiten.Vertex
	contactMaskInds    []uint16
	contactDirectVerts []ebiten.Vertex
	contactDirectInds  []uint16

	shadowBatch    *BatchRenderer
	shadowTarget   *ebiten.Image
	blurTarget     *ebiten.Image
	blurTempTarget *ebiten.Image

	blurShaderOpts  ebiten.DrawTrianglesShaderOptions
	compositeOpts   ebiten.DrawImageOptions
	dirHAny         any
	dirVAny         any
	tintPass1Any    any
	tintPass2       []float32
	tintPass2Any    any
	cachedRadius    float32
	cachedRadiusAny any

	initialized bool
}

// NewShadowRenderSystem creates a new ShadowRenderSystem with pre-allocated rendering buffers.
func NewShadowRenderSystem(opts ...ShadowOption) *ShadowRenderSystem {
	tintPass2 := []float32{1, 1, 1, 1}
	s := &ShadowRenderSystem{
		blurDownsample:     defaultBlurDownsample,
		globalOpacity:      defaultShadowOpacity,
		globalColor:        color.RGBA{R: 255, G: 255, B: 255, A: 255},
		frustumMargin:      defaultShadowCullMargin,
		quadIndices:        [6]uint16{0, 1, 2, 0, 2, 3},
		contactMaskVerts:   make([]ebiten.Vertex, 0, 512),
		contactMaskInds:    make([]uint16, 0, 768),
		contactDirectVerts: make([]ebiten.Vertex, 0, 512),
		contactDirectInds:  make([]uint16, 0, 768),
		dirHAny:            []float32{1, 0},
		dirVAny:            []float32{0, 1},
		tintPass1Any:       []float32{1, 1, 1, 1},
		tintPass2:          tintPass2,
		tintPass2Any:       tintPass2,
		cachedRadius:       -1,
	}
	s.blurShaderOpts.Uniforms = make(map[string]any, 4)
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// SetComposited enables or disables single-pass offscreen shadow mask compositing.
func (s *ShadowRenderSystem) SetComposited(enabled bool) {
	s.composited = enabled
	if !enabled {
		s.Dispose()
	}
}

// SetBlurRadius sets the 2-pass Gaussian blur radius in pixels.
// Setting a positive radius implicitly enables composited mode.
func (s *ShadowRenderSystem) SetBlurRadius(radius float64) {
	if radius < 0 {
		radius = 0
	}
	s.blurRadius = radius
	if radius > 0 {
		s.composited = true
	} else {
		if s.blurTarget != nil {
			s.blurTarget.Deallocate()
			s.blurTarget = nil
		}
		if s.blurTempTarget != nil {
			s.blurTempTarget.Deallocate()
			s.blurTempTarget = nil
		}
	}
}

// SetBlurDownsample sets the resolution scale factor for offscreen shadow blur passes,
// clamped to [0.25, 1.0].
func (s *ShadowRenderSystem) SetBlurDownsample(scale float64) {
	s.blurDownsample = Clamp(scale, minBlurDownsample, maxBlurDownsample)
	if s.blurDownsample >= 1.0 && s.blurTempTarget != nil {
		s.blurTempTarget.Deallocate()
		s.blurTempTarget = nil
	}
}

// SetGlobalOpacity sets the uniform opacity used when compositing the offscreen shadow mask.
func (s *ShadowRenderSystem) SetGlobalOpacity(opacity float64) {
	s.globalOpacity = Clamp(opacity, 0.0, 1.0)
}

// IsComposited reports whether offscreen shadow mask compositing is enabled.
func (s *ShadowRenderSystem) IsComposited() bool {
	return s.composited
}

// BlurRadius returns the current Gaussian blur radius in pixels.
func (s *ShadowRenderSystem) BlurRadius() float64 {
	return s.blurRadius
}

// BlurDownsample returns the current offscreen shadow blur resolution scale factor.
func (s *ShadowRenderSystem) BlurDownsample() float64 {
	return s.blurDownsample
}

// Dispose releases all offscreen GPU render targets held by the system.
func (s *ShadowRenderSystem) Dispose() {
	if s.shadowTarget != nil {
		s.shadowTarget.Deallocate()
		s.shadowTarget = nil
	}
	if s.blurTarget != nil {
		s.blurTarget.Deallocate()
		s.blurTarget = nil
	}
	if s.blurTempTarget != nil {
		s.blurTempTarget.Deallocate()
		s.blurTempTarget = nil
	}
	s.blurShaderOpts.Images[0] = nil
}

// Initialize binds the system's ECS filters, builder, resource handles, and layout listener once.
func (s *ShadowRenderSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	s.shadowFilter = teishoku.NewFilter2[TransformComponent, ShadowComponent](w)
	s.lightFilter = teishoku.NewFilter[ShadowLightComponent](w)
	s.spriteBuilder = teishoku.NewBuilder[SpriteComponent](w)
	s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())

	Subscribe(w, s.onEngineLayoutChanged)
	s.initialized = true
}

func (s *ShadowRenderSystem) onEngineLayoutChanged(evt EngineLayoutChangedEvent) {
	if s.composited {
		s.ensureTargets(evt.Width, evt.Height, s.blurRadius > 0)
	}
}

func (s *ShadowRenderSystem) ensureShadowBatch() {
	if s.shadowBatch == nil {
		s.shadowBatch = NewBatchRenderer()
	}
}

func (s *ShadowRenderSystem) ensureTargets(width, height int, needBlur bool) {
	if width <= 0 || height <= 0 {
		return
	}
	if s.shadowTarget != nil {
		b := s.shadowTarget.Bounds()
		if b.Dx() != width || b.Dy() != height {
			s.shadowTarget.Deallocate()
			s.shadowTarget = nil
		}
	}
	if s.shadowTarget == nil {
		s.shadowTarget = ebiten.NewImage(width, height)
	}

	if !needBlur {
		if s.blurTarget != nil {
			s.blurTarget.Deallocate()
			s.blurTarget = nil
		}
		if s.blurTempTarget != nil {
			s.blurTempTarget.Deallocate()
			s.blurTempTarget = nil
		}
		return
	}

	scale := Clamp(s.blurDownsample, minBlurDownsample, maxBlurDownsample)
	blurW, blurH := width, height
	useDownsample := scale < 1.0
	if useDownsample {
		blurW = max(1, int(float64(width)*scale))
		blurH = max(1, int(float64(height)*scale))
	}

	if s.blurTarget != nil {
		b := s.blurTarget.Bounds()
		if b.Dx() != blurW || b.Dy() != blurH {
			s.blurTarget.Deallocate()
			s.blurTarget = nil
		}
	}
	if s.blurTarget == nil {
		s.blurTarget = ebiten.NewImage(blurW, blurH)
	}

	if !useDownsample {
		if s.blurTempTarget != nil {
			s.blurTempTarget.Deallocate()
			s.blurTempTarget = nil
		}
		return
	}

	if s.blurTempTarget != nil {
		b := s.blurTempTarget.Bounds()
		if b.Dx() != blurW || b.Dy() != blurH {
			s.blurTempTarget.Deallocate()
			s.blurTempTarget = nil
		}
	}
	if s.blurTempTarget == nil {
		s.blurTempTarget = ebiten.NewImage(blurW, blurH)
	}
}

func (s *ShadowRenderSystem) beginMask(rdr *BatchRenderer, screenW, screenH int, useBlur bool) bool {
	s.ensureTargets(screenW, screenH, useBlur)
	if s.shadowTarget == nil {
		return false
	}
	s.ensureShadowBatch()
	s.shadowTarget.Fill(color.RGBA{})
	s.shadowBatch.Begin(s.shadowTarget)
	if vm, hasVM := rdr.GetViewMatrix(); hasVM {
		s.shadowBatch.SetViewMatrix(vm)
	} else {
		s.shadowBatch.ResetViewMatrix()
	}
	return true
}

func appendContactQuad(verts []ebiten.Vertex, inds []uint16, quad *[4]ebiten.Vertex) ([]ebiten.Vertex, []uint16) {
	base := uint16(len(verts))
	verts = append(verts, quad[0], quad[1], quad[2], quad[3])
	inds = append(inds, base, base+1, base+2, base, base+2, base+3)
	return verts, inds
}

// Draw renders all enabled ShadowComponents in the world.
func (s *ShadowRenderSystem) Draw(w *teishoku.World, rdr *BatchRenderer) {
	if w == nil || rdr == nil {
		return
	}
	if !s.initialized {
		s.Initialize(w)
	}
	screen := rdr.GetScreen()
	if screen == nil {
		return
	}
	screenBounds := screen.Bounds()
	screenW, screenH := screenBounds.Dx(), screenBounds.Dy()
	if screenW <= 0 || screenH <= 0 {
		return
	}

	tm := s.tmHandle.Get()
	if tm == nil {
		tm = GetTextureManager(w)
		if tm == nil {
			return
		}
		s.tmHandle = teishoku.UseResource[TextureManager](w.Resources())
	}

	cam := s.camHandle.Get()
	if cam == nil {
		if c := GetActiveCamera(w); c != nil {
			s.camHandle = teishoku.UseResource[ActiveCamera](w.Resources())
			cam = c
		}
	}
	useCull := cam != nil && cam.HasCamera

	var light ShadowLightComponent
	var hasLight bool
	s.lightFilter.EachChunk(func(lights []ShadowLightComponent) {
		if hasLight {
			return
		}
		for i := range lights {
			if lights[i].Enabled {
				light = lights[i]
				hasLight = true
				return
			}
		}
	})

	useComposited := s.composited
	useBlur := useComposited && s.blurRadius > 0
	maskBegun := false

	s.contactMaskVerts = s.contactMaskVerts[:0]
	s.contactMaskInds = s.contactMaskInds[:0]
	s.contactDirectVerts = s.contactDirectVerts[:0]
	s.contactDirectInds = s.contactDirectInds[:0]

	chunks := s.shadowFilter.Chunks()
	for chunks.Next() {
		ents := chunks.Entities()
		transforms, shadows := chunks.Get()
		for i := range shadows {
			sh := &shadows[i]
			if !sh.Enabled {
				continue
			}

			angle := sh.Angle
			length := sh.Length
			col := sh.Color
			opacity := sh.Opacity
			if sh.UseGlobalLight && hasLight {
				angle = light.Angle
				length = light.Length
				col = light.Color
				opacity = light.Opacity
			}

			t := &transforms[i]
			if useCull {
				scaleMax := math.Max(1.0, math.Max(math.Abs(t.Scale.X), math.Abs(t.Scale.Y)))
				ext := math.Max(256.0, math.Max(math.Abs(length), math.Abs(angle))*256.0) * scaleMax
				coarseMargin := s.frustumMargin + ext +
					math.Abs(sh.Offset.X) + math.Abs(sh.Offset.Y) + math.Abs(sh.Elevation) +
					math.Abs(t.Origin.X) + math.Abs(t.Origin.Y)
				if !cam.IsPointVisible(Vector(t.Position), coarseMargin) {
					continue
				}
			}

			toMask := useComposited && !sh.PreserveColor
			drawOpacity := opacity
			drawCol := col
			if toMask {
				drawOpacity = 1.0
				if s.hasGlobalColor {
					drawCol = color.RGBA{R: 255, G: 255, B: 255, A: 255}
				}
			}

			sp := s.spriteBuilder.Get(ents[i])

			var tex *ebiten.Image
			switch sh.Type {
			case ShadowTypeContact:
				tex = contactShadowTexture()
				tb := tex.Bounds()
				if !buildContactShadowVertices(
					&s.quadVerts, t, sp, sh,
					float32(tb.Min.X), float32(tb.Min.Y), float32(tb.Max.X), float32(tb.Max.Y),
					drawCol, drawOpacity,
				) {
					continue
				}
			case ShadowTypeProjected:
				if sp == nil || sp.Width <= 0 || sp.Height <= 0 {
					continue
				}
				tex = tm.Get(sp.TextureID)
				if tex == nil {
					continue
				}
				srcMinX, srcMinY, srcMaxX, srcMaxY := resolveSpriteSourceBounds(sp, tex)
				buildProjectedShadowVertices(
					&s.quadVerts, t, sp, sh, angle, length,
					srcMinX, srcMinY, srcMaxX, srcMaxY,
					drawCol, drawOpacity, toMask,
				)
			case ShadowTypeDrop:
				if sp == nil || sp.Width <= 0 || sp.Height <= 0 {
					continue
				}
				tex = tm.Get(sp.TextureID)
				if tex == nil {
					continue
				}
				srcMinX, srcMinY, srcMaxX, srcMaxY := resolveSpriteSourceBounds(sp, tex)
				buildDropShadowVertices(
					&s.quadVerts, t, sp, sh, angle,
					srcMinX, srcMinY, srcMaxX, srcMaxY,
					drawCol, drawOpacity, sh.PreserveColor,
				)
			default:
				continue
			}

			if useCull {
				minX := float64(min(min(s.quadVerts[0].DstX, s.quadVerts[1].DstX), min(s.quadVerts[2].DstX, s.quadVerts[3].DstX)))
				maxX := float64(max(max(s.quadVerts[0].DstX, s.quadVerts[1].DstX), max(s.quadVerts[2].DstX, s.quadVerts[3].DstX)))
				minY := float64(min(min(s.quadVerts[0].DstY, s.quadVerts[1].DstY), min(s.quadVerts[2].DstY, s.quadVerts[3].DstY)))
				maxY := float64(max(max(s.quadVerts[0].DstY, s.quadVerts[1].DstY), max(s.quadVerts[2].DstY, s.quadVerts[3].DstY)))
				bounds := Rectangle{
					Min: Vector{X: minX - s.frustumMargin, Y: minY - s.frustumMargin},
					Max: Vector{X: maxX + s.frustumMargin, Y: maxY + s.frustumMargin},
				}
				if !cam.IsVisible(bounds) {
					continue
				}
			}

			if sh.Type == ShadowTypeContact {
				if toMask {
					if len(s.contactMaskVerts)+4 > maxContactBatchVerts {
						if !maskBegun {
							maskBegun = s.beginMask(rdr, screenW, screenH, useBlur)
						}
						if maskBegun {
							s.shadowBatch.AddCustomMeshes(s.contactMaskVerts, s.contactMaskInds, tex)
						}
						s.contactMaskVerts = s.contactMaskVerts[:0]
						s.contactMaskInds = s.contactMaskInds[:0]
					}
					s.contactMaskVerts, s.contactMaskInds = appendContactQuad(s.contactMaskVerts, s.contactMaskInds, &s.quadVerts)
				} else {
					if len(s.contactDirectVerts)+4 > maxContactBatchVerts {
						rdr.AddCustomMeshes(s.contactDirectVerts, s.contactDirectInds, tex)
						s.contactDirectVerts = s.contactDirectVerts[:0]
						s.contactDirectInds = s.contactDirectInds[:0]
					}
					s.contactDirectVerts, s.contactDirectInds = appendContactQuad(s.contactDirectVerts, s.contactDirectInds, &s.quadVerts)
				}
				continue
			}

			if toMask {
				if !maskBegun {
					maskBegun = s.beginMask(rdr, screenW, screenH, useBlur)
					if !maskBegun {
						continue
					}
				}
				s.shadowBatch.AddCustomMeshes(s.quadVerts[:], s.quadIndices[:], tex)
			} else {
				rdr.AddCustomMeshes(s.quadVerts[:], s.quadIndices[:], tex)
			}
		}
	}

	if len(s.contactMaskVerts) > 0 {
		if !maskBegun {
			maskBegun = s.beginMask(rdr, screenW, screenH, useBlur)
		}
		if maskBegun {
			s.shadowBatch.AddCustomMeshes(s.contactMaskVerts, s.contactMaskInds, contactShadowTexture())
		}
		s.contactMaskVerts = s.contactMaskVerts[:0]
		s.contactMaskInds = s.contactMaskInds[:0]
	}
	if len(s.contactDirectVerts) > 0 {
		rdr.AddCustomMeshes(s.contactDirectVerts, s.contactDirectInds, contactShadowTexture())
		s.contactDirectVerts = s.contactDirectVerts[:0]
		s.contactDirectInds = s.contactDirectInds[:0]
	}

	if maskBegun {
		s.shadowBatch.Flush()
		rdr.Flush()
		s.compositeMask(screen, screenW, screenH, useBlur)
	}
}

func (s *ShadowRenderSystem) compositeMask(screen *ebiten.Image, screenW, screenH int, useBlur bool) {
	a := float32(Clamp(s.globalOpacity, 0.0, 1.0))
	var tr, tg, tb, ta float32
	if s.hasGlobalColor {
		const inv255 = float32(1.0 / 255.0)
		ca := float32(s.globalColor.A) * inv255
		ta = a * ca
		tr = float32(s.globalColor.R) * inv255 * ta
		tg = float32(s.globalColor.G) * inv255 * ta
		tb = float32(s.globalColor.B) * inv255 * ta
	} else {
		tr, tg, tb, ta = a, a, a, a
	}

	screenBounds := screen.Bounds()
	if useBlur && s.blurTarget != nil {
		shader := getShadowBlurShader()
		scale := Clamp(s.blurDownsample, minBlurDownsample, maxBlurDownsample)
		if scale < 1.0 && s.blurTempTarget != nil {
			tempBounds := s.blurTempTarget.Bounds()
			blurW, blurH := tempBounds.Dx(), tempBounds.Dy()

			s.blurTempTarget.Fill(color.RGBA{})
			s.compositeOpts.GeoM.Reset()
			s.compositeOpts.GeoM.Scale(float64(blurW)/float64(screenW), float64(blurH)/float64(screenH))
			s.compositeOpts.ColorScale.Reset()
			s.compositeOpts.Filter = ebiten.FilterLinear
			s.blurTempTarget.DrawImage(s.shadowTarget, &s.compositeOpts)

			bw := float32(blurW)
			bh := float32(blurH)
			s.screenVerts[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
			s.screenVerts[1] = ebiten.Vertex{DstX: bw, DstY: 0, SrcX: bw, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
			s.screenVerts[2] = ebiten.Vertex{DstX: bw, DstY: bh, SrcX: bw, SrcY: bh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
			s.screenVerts[3] = ebiten.Vertex{DstX: 0, DstY: bh, SrcX: 0, SrcY: bh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}

			radius32 := float32(s.blurRadius * scale)
			if radius32 != s.cachedRadius || s.cachedRadiusAny == nil {
				s.cachedRadius = radius32
				s.cachedRadiusAny = radius32
				s.blurShaderOpts.Uniforms["Radius"] = s.cachedRadiusAny
			}

			s.blurTarget.Fill(color.RGBA{})
			s.blurShaderOpts.Images[0] = s.blurTempTarget
			s.blurShaderOpts.Uniforms["Direction"] = s.dirHAny
			s.blurShaderOpts.Uniforms["Tint"] = s.tintPass1Any
			s.blurTarget.DrawTrianglesShader(s.screenVerts[:], s.quadIndices[:], shader, &s.blurShaderOpts)

			s.blurTempTarget.Fill(color.RGBA{})
			s.blurShaderOpts.Images[0] = s.blurTarget
			s.blurShaderOpts.Uniforms["Direction"] = s.dirVAny
			s.blurShaderOpts.Uniforms["Tint"] = s.tintPass1Any
			s.blurTempTarget.DrawTrianglesShader(s.screenVerts[:], s.quadIndices[:], shader, &s.blurShaderOpts)
			s.blurShaderOpts.Images[0] = nil

			s.compositeOpts.GeoM.Reset()
			s.compositeOpts.GeoM.Scale(float64(screenW)/float64(blurW), float64(screenH)/float64(blurH))
			if screenBounds.Min.X != 0 || screenBounds.Min.Y != 0 {
				s.compositeOpts.GeoM.Translate(float64(screenBounds.Min.X), float64(screenBounds.Min.Y))
			}
			s.compositeOpts.ColorScale.Reset()
			s.compositeOpts.ColorScale.Scale(tr, tg, tb, ta)
			s.compositeOpts.Filter = ebiten.FilterLinear
			screen.DrawImage(s.blurTempTarget, &s.compositeOpts)
			return
		}

		s.blurTarget.Fill(color.RGBA{})

		fw := float32(screenW)
		fh := float32(screenH)
		s.screenVerts[0] = ebiten.Vertex{DstX: 0, DstY: 0, SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
		s.screenVerts[1] = ebiten.Vertex{DstX: fw, DstY: 0, SrcX: fw, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
		s.screenVerts[2] = ebiten.Vertex{DstX: fw, DstY: fh, SrcX: fw, SrcY: fh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}
		s.screenVerts[3] = ebiten.Vertex{DstX: 0, DstY: fh, SrcX: 0, SrcY: fh, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1}

		radius32 := float32(s.blurRadius)
		if radius32 != s.cachedRadius || s.cachedRadiusAny == nil {
			s.cachedRadius = radius32
			s.cachedRadiusAny = radius32
			s.blurShaderOpts.Uniforms["Radius"] = s.cachedRadiusAny
		}
		s.blurShaderOpts.Images[0] = s.shadowTarget
		s.blurShaderOpts.Uniforms["Direction"] = s.dirHAny
		s.blurShaderOpts.Uniforms["Tint"] = s.tintPass1Any
		s.blurTarget.DrawTrianglesShader(s.screenVerts[:], s.quadIndices[:], shader, &s.blurShaderOpts)

		if screenBounds.Min.X != 0 || screenBounds.Min.Y != 0 {
			dx0 := float32(screenBounds.Min.X)
			dy0 := float32(screenBounds.Min.Y)
			dx1 := float32(screenBounds.Max.X)
			dy1 := float32(screenBounds.Max.Y)
			s.screenVerts[0].DstX, s.screenVerts[0].DstY = dx0, dy0
			s.screenVerts[1].DstX, s.screenVerts[1].DstY = dx1, dy0
			s.screenVerts[2].DstX, s.screenVerts[2].DstY = dx1, dy1
			s.screenVerts[3].DstX, s.screenVerts[3].DstY = dx0, dy1
		}

		s.tintPass2[0] = tr
		s.tintPass2[1] = tg
		s.tintPass2[2] = tb
		s.tintPass2[3] = ta
		s.blurShaderOpts.Images[0] = s.blurTarget
		s.blurShaderOpts.Uniforms["Direction"] = s.dirVAny
		s.blurShaderOpts.Uniforms["Tint"] = s.tintPass2Any
		screen.DrawTrianglesShader(s.screenVerts[:], s.quadIndices[:], shader, &s.blurShaderOpts)
		s.blurShaderOpts.Images[0] = nil
		return
	}

	s.compositeOpts.GeoM.Reset()
	if screenBounds.Min.X != 0 || screenBounds.Min.Y != 0 {
		s.compositeOpts.GeoM.Translate(float64(screenBounds.Min.X), float64(screenBounds.Min.Y))
	}
	s.compositeOpts.ColorScale.Reset()
	s.compositeOpts.ColorScale.Scale(tr, tg, tb, ta)
	s.compositeOpts.Filter = ebiten.FilterNearest
	screen.DrawImage(s.shadowTarget, &s.compositeOpts)
}

func premultipliedShadowRGBA(col color.RGBA, alpha float64, preserveColor bool) (float32, float32, float32, float32) {
	a := float32(Clamp(alpha, 0.0, 1.0))
	if preserveColor {
		return a, a, a, a
	}
	const inv255 = float32(1.0 / 255.0)
	ca := a * float32(col.A) * inv255
	return float32(col.R) * inv255 * ca,
		float32(col.G) * inv255 * ca,
		float32(col.B) * inv255 * ca,
		ca
}

// buildProjectedShadowVertices populates dst with 4 vertices representing a projected
// cast shadow (or reflection) anchored at the caster's base.
func buildProjectedShadowVertices(
	dst *[4]ebiten.Vertex,
	t *TransformComponent,
	sp *SpriteComponent,
	sh *ShadowComponent,
	angle, length float64,
	srcMinX, srcMinY, srcMaxX, srcMaxY float32,
	col color.RGBA,
	opacity float64,
	isMask bool,
) {
	shScaleX := sh.Scale.X
	if shScaleX == 0 {
		shScaleX = 1
	}
	shScaleY := sh.Scale.Y
	if shScaleY == 0 {
		shScaleY = 1
	}

	dstW := float64(sp.Width)
	dstH := float64(sp.Height)
	absESX := math.Abs(t.Scale.X)
	absESY := math.Abs(t.Scale.Y)

	lx0 := -t.Offset.X * t.Scale.X
	lx1 := (dstW - t.Offset.X) * t.Scale.X
	ly := (dstH - t.Offset.Y) * t.Scale.Y

	sMinX, sMaxX := srcMinX, srcMaxX
	if lx0 > lx1 {
		lx0, lx1 = lx1, lx0
		sMinX, sMaxX = sMaxX, sMinX
	}

	if shScaleX != 1 {
		midX := (lx0 + lx1) * 0.5
		halfW := (lx1 - lx0) * 0.5 * shScaleX
		lx0 = midX - halfW
		lx1 = midX + halfW
	}

	var bx0, by0, bx1, by1 float64
	if t.Rotation == 0 {
		bx0, by0 = lx0, ly
		bx1, by1 = lx1, ly
	} else {
		sin, cos := math.Sincos(t.Rotation)
		bx0 = lx0*cos - ly*sin
		by0 = lx0*sin + ly*cos
		bx1 = lx1*cos - ly*sin
		by1 = lx1*sin + ly*cos
	}

	elevX := sh.Elevation * angle
	elevY := sh.Elevation * 0.5
	tx := t.Position.X - t.Origin.X + sh.Offset.X + elevX
	ty := t.Position.Y - t.Origin.Y + sh.Offset.Y + elevY
	bx0 += tx
	by0 += ty
	bx1 += tx
	by1 += ty

	skewX := dstW * angle * absESX * shScaleX
	projY := dstH * length * absESY * shScaleY
	tx0 := bx0 + skewX
	ty0 := by0 + projY
	tx1 := bx1 + skewX
	ty1 := by1 + projY

	baseAlpha := Clamp(opacity, 0.0, 1.0)
	if isMask {
		baseAlpha = 1.0
	}
	tipAlpha := baseAlpha
	if sh.FadeOut {
		tipAlpha = baseAlpha * Clamp(sh.FadeAlpha, 0.0, 1.0)
	}

	preserve := sh.PreserveColor && !isMask
	baseR, baseG, baseB, baseA := premultipliedShadowRGBA(col, baseAlpha, preserve)
	tipR, tipG, tipB, tipA := premultipliedShadowRGBA(col, tipAlpha, preserve)

	dst[0] = ebiten.Vertex{
		DstX: float32(bx0), DstY: float32(by0),
		SrcX: sMinX, SrcY: srcMaxY,
		ColorR: baseR, ColorG: baseG, ColorB: baseB, ColorA: baseA,
	}
	dst[1] = ebiten.Vertex{
		DstX: float32(bx1), DstY: float32(by1),
		SrcX: sMaxX, SrcY: srcMaxY,
		ColorR: baseR, ColorG: baseG, ColorB: baseB, ColorA: baseA,
	}
	dst[2] = ebiten.Vertex{
		DstX: float32(tx1), DstY: float32(ty1),
		SrcX: sMaxX, SrcY: srcMinY,
		ColorR: tipR, ColorG: tipG, ColorB: tipB, ColorA: tipA,
	}
	dst[3] = ebiten.Vertex{
		DstX: float32(tx0), DstY: float32(ty0),
		SrcX: sMinX, SrcY: srcMinY,
		ColorR: tipR, ColorG: tipG, ColorB: tipB, ColorA: tipA,
	}
}

// buildDropShadowVertices populates dst with 4 vertices representing an offset
// silhouette copy of the entity's sprite.
func buildDropShadowVertices(
	dst *[4]ebiten.Vertex,
	t *TransformComponent,
	sp *SpriteComponent,
	sh *ShadowComponent,
	angle float64,
	srcMinX, srcMinY, srcMaxX, srcMaxY float32,
	col color.RGBA,
	opacity float64,
	preserveColor bool,
) {
	shScaleX := sh.Scale.X
	if shScaleX == 0 {
		shScaleX = 1
	}
	shScaleY := sh.Scale.Y
	if shScaleY == 0 {
		shScaleY = 1
	}

	scaleX := t.Scale.X * shScaleX
	scaleY := t.Scale.Y * shScaleY
	dstW := float64(sp.Width)
	dstH := float64(sp.Height)

	x0 := -t.Offset.X * scaleX
	y0 := -t.Offset.Y * scaleY
	x1 := (dstW - t.Offset.X) * scaleX
	y1 := (dstH - t.Offset.Y) * scaleY

	sMinX, sMaxX := srcMinX, srcMaxX
	if x0 > x1 {
		x0, x1 = x1, x0
		sMinX, sMaxX = sMaxX, sMinX
	}

	elevX := sh.Elevation * angle
	elevY := sh.Elevation * 0.5
	tx := t.Position.X - t.Origin.X + sh.Offset.X + elevX
	ty := t.Position.Y - t.Origin.Y + sh.Offset.Y + elevY

	var p0x, p0y, p1x, p1y, p2x, p2y, p3x, p3y float64
	if t.Rotation == 0 {
		p0x, p0y = x0+tx, y0+ty
		p1x, p1y = x1+tx, y0+ty
		p2x, p2y = x1+tx, y1+ty
		p3x, p3y = x0+tx, y1+ty
	} else {
		sin, cos := math.Sincos(t.Rotation)
		x0c, x0s := x0*cos, x0*sin
		x1c, x1s := x1*cos, x1*sin
		y0c, y0s := y0*cos, y0*sin
		y1c, y1s := y1*cos, y1*sin

		p0x, p0y = x0c-y0s+tx, x0s+y0c+ty
		p1x, p1y = x1c-y0s+tx, x1s+y0c+ty
		p2x, p2y = x1c-y1s+tx, x1s+y1c+ty
		p3x, p3y = x0c-y1s+tx, x0s+y1c+ty
	}

	cr, cg, cb, ca := premultipliedShadowRGBA(col, opacity, preserveColor)
	dst[0] = ebiten.Vertex{DstX: float32(p0x), DstY: float32(p0y), SrcX: sMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[1] = ebiten.Vertex{DstX: float32(p1x), DstY: float32(p1y), SrcX: sMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[2] = ebiten.Vertex{DstX: float32(p2x), DstY: float32(p2y), SrcX: sMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[3] = ebiten.Vertex{DstX: float32(p3x), DstY: float32(p3y), SrcX: sMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
}

// buildContactShadowVertices populates dst with 4 vertices for an elliptical radial
// contact shadow centered at the caster's ground base. Returns false if the shadow has zero area.
func buildContactShadowVertices(
	dst *[4]ebiten.Vertex,
	t *TransformComponent,
	sp *SpriteComponent,
	sh *ShadowComponent,
	srcMinX, srcMinY, srcMaxX, srcMaxY float32,
	col color.RGBA,
	opacity float64,
) bool {
	shScaleX := sh.Scale.X
	if shScaleX == 0 {
		shScaleX = 1
	}
	shScaleY := sh.Scale.Y
	if shScaleY == 0 {
		shScaleY = 1
	}

	rx := sh.RadiusX * shScaleX
	ry := sh.RadiusY * shScaleY
	if rx <= 0 && sp != nil && sp.Width > 0 {
		rx = float64(sp.Width) * 0.45 * math.Abs(t.Scale.X) * shScaleX
	}
	if ry <= 0 && rx > 0 {
		ry = rx * 0.45
	}
	if rx <= 0 || ry <= 0 {
		return false
	}

	elev := math.Max(0.0, sh.Elevation)
	elevScale := 1.0 / (1.0 + elev*0.015)
	elevAlpha := 1.0 / (1.0 + elev*0.02)
	rx *= elevScale
	ry *= elevScale
	effOpacity := Clamp(opacity, 0.0, 1.0) * elevAlpha

	var lx, ly float64
	if sp != nil {
		lx = (float64(sp.Width)*0.5 - t.Offset.X) * t.Scale.X
		ly = (float64(sp.Height) - t.Offset.Y) * t.Scale.Y
	} else {
		lx = -t.Offset.X * t.Scale.X
		ly = -t.Offset.Y * t.Scale.Y
	}
	if t.Rotation != 0 {
		sin, cos := math.Sincos(t.Rotation)
		lx, ly = lx*cos-ly*sin, lx*sin+ly*cos
	}

	cx := lx + (t.Position.X - t.Origin.X) + sh.Offset.X
	cy := ly + (t.Position.Y - t.Origin.Y) + sh.Offset.Y

	x0 := float32(cx - rx)
	y0 := float32(cy - ry)
	x1 := float32(cx + rx)
	y1 := float32(cy + ry)

	cr, cg, cb, ca := premultipliedShadowRGBA(col, effOpacity, false)
	dst[0] = ebiten.Vertex{DstX: x0, DstY: y0, SrcX: srcMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[1] = ebiten.Vertex{DstX: x1, DstY: y0, SrcX: srcMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[2] = ebiten.Vertex{DstX: x1, DstY: y1, SrcX: srcMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	dst[3] = ebiten.Vertex{DstX: x0, DstY: y1, SrcX: srcMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
	return true
}
