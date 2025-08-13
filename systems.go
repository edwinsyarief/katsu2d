package katsu2d

import (
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/katsu2d/utils"
)

// --- SYSTEMS ---

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Update(*World, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Draw(*World, *BatchRenderer)
}

// --- RENDERER ---

// BatchRenderer batches draw calls for performance.
type BatchRenderer struct {
	screen       *ebiten.Image
	vertices     []ebiten.Vertex
	indices      []uint16
	currentImage *ebiten.Image
}

// NewBatchRenderer creates a new batch renderer.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{
		vertices: make([]ebiten.Vertex, 0, 4096),
		indices:  make([]uint16, 0, 6144),
	}
}

// GetScreen returns the current screen image being rendered to.
func (self *BatchRenderer) GetScreen() *ebiten.Image {
	return self.screen
}

// Begin prepares the renderer for a new frame.
func (self *BatchRenderer) Begin(screen *ebiten.Image) {
	self.screen = screen
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// Flush draws the current batch.
func (self *BatchRenderer) Flush() {
	if len(self.vertices) == 0 {
		return
	}
	self.screen.DrawTriangles(self.vertices, self.indices, self.currentImage, nil)
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// AddVertices adds custom vertices and indices to the batch.
func (self *BatchRenderer) AddVertices(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)
	self.vertices = append(self.vertices, verts...)
	for _, i := range inds {
		self.indices = append(self.indices, uint16(offset)+i)
	}
}

// DrawQuad draws a quad (sprite) with specified source rectangle and destination size.
func (self *BatchRenderer) DrawQuad(pos, scale, offset, origin ebimath.Vector, rotation float64, img *ebiten.Image, clr color.RGBA, srcMinX, srcMinY, srcMaxX, srcMaxY float32, destW, destH float64) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img

	w, h := destW*scale.X, destH*scale.Y
	ox, oy := origin.X, origin.Y

	p0 := ebimath.V(-ox, -oy)
	p1 := ebimath.V(w-ox, -oy)
	p2 := ebimath.V(w-ox, h-oy)
	p3 := ebimath.V(-ox, h-oy)

	if rotation != 0 {
		c, s := math.Cos(rotation), math.Sin(rotation)
		p0 = ebimath.V(p0.X*c-p0.Y*s, p0.X*s+p0.Y*c)
		p1 = ebimath.V(p1.X*c-p1.Y*s, p1.X*s+p1.Y*c)
		p2 = ebimath.V(p2.X*c-p2.Y*s, p2.X*s+p2.Y*c)
		p3 = ebimath.V(p3.X*c-p3.Y*s, p3.X*s+p3.Y*c)
	}

	p0 = p0.Add(offset)
	p1 = p1.Add(offset)
	p2 = p2.Add(offset)
	p3 = p3.Add(offset)

	p0 = p0.Add(pos)
	p1 = p1.Add(pos)
	p2 = p2.Add(pos)
	p3 = p3.Add(pos)

	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255

	vertIndex := len(self.vertices)
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p0.X)), DstY: utils.AdjustDestinationPixel(float32(p0.Y)), SrcX: srcMinX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p1.X)), DstY: utils.AdjustDestinationPixel(float32(p1.Y)), SrcX: srcMaxX, SrcY: srcMinY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p2.X)), DstY: utils.AdjustDestinationPixel(float32(p2.Y)), SrcX: srcMaxX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p3.X)), DstY: utils.AdjustDestinationPixel(float32(p3.Y)), SrcX: srcMinX, SrcY: srcMaxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	self.indices = append(self.indices, uint16(vertIndex), uint16(vertIndex+1), uint16(vertIndex+2), uint16(vertIndex), uint16(vertIndex+2), uint16(vertIndex+3))
}

// DrawTriangleStrip draws a triangle strip.
func (self *BatchRenderer) DrawTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)
	self.vertices = append(self.vertices, verts...)
	for i := 0; i < len(verts)-2; i++ {
		a := uint16(offset + i)
		bb := uint16(offset + i + 1)
		c := uint16(offset + i + 2)
		if i%2 == 0 {
			self.indices = append(self.indices, a, bb, c)
		} else {
			self.indices = append(self.indices, a, c, bb)
		}
	}
}

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

// NewTweenSystem creates a new TweenSystem.
func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

// Update updates all standalone tweens and sequences in the world.
func (self *TweenSystem) Update(world *World, dt float64) {
	// Standalone tweens
	entities := world.Query(CTTween)
	for _, e := range entities {
		twAny, _ := world.GetComponent(e, CTTween)
		tw := twAny.(*tween.Tween)
		tw.Update(float32(dt))
	}

	// Standalone Sequences
	entities = world.Query(CTSequence)
	for _, e := range entities {
		seqAny, _ := world.GetComponent(e, CTSequence)
		seq := seqAny.(*tween.Sequence)
		seq.Update(float32(dt))
	}
}

// AnimationSystem updates animations.
type AnimationSystem struct{}

// NewAnimationSystem creates a new AnimationSystem.
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

// Update advances all active animations in the world by the given delta time.
func (self *AnimationSystem) Update(world *World, dt float64) {
	entities := world.Query(CTAnimation, CTSprite)
	for _, e := range entities {
		animAny, _ := world.GetComponent(e, CTAnimation)
		anim := animAny.(*AnimationComponent)
		sprAny, _ := world.GetComponent(e, CTSprite)
		spr := sprAny.(*SpriteComponent)

		if !anim.Active || len(anim.Frames) == 0 {
			continue
		}
		anim.Elapsed += dt
		if anim.Elapsed >= anim.Speed {
			anim.Elapsed -= anim.Speed
			nf := len(anim.Frames)
			switch anim.Mode {
			case AnimOnce:
				if anim.Current+1 >= nf {
					anim.Current = nf - 1
					anim.Active = false
				} else {
					anim.Current++
				}
			case AnimLoop:
				anim.Current++
				anim.Current %= nf
			case AnimBoomerang:
				// --- FIX: Handle single-frame boomerang gracefully ---
				if nf > 1 {
					if anim.Direction {
						anim.Current++
						if anim.Current >= nf-1 {
							anim.Current = nf - 1
							anim.Direction = false
						}
					} else {
						anim.Current--
						if anim.Current < 0 {
							anim.Current = 0
							anim.Direction = true
						}
					}
				} else {
					anim.Current = 0
					anim.Active = false
				}
			}
			frame := anim.Frames[anim.Current]
			spr.SrcX = float32(frame.Min.X)
			spr.SrcY = float32(frame.Min.Y)
			spr.SrcW = float32(frame.Dx())
			spr.SrcH = float32(frame.Dy())
		}
	}
}

// FadeOverlaySystem manages fade overlays.
type FadeOverlaySystem struct{}

// NewFadeOverlaySystem creates a new FadeOverlaySystem.
func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{}
}

// Update updates all fade overlays in the world.
func (self *FadeOverlaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Update(dt)
	}
}

// Draw renders all fade overlays to the screen.
func (self *FadeOverlaySystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Draw(renderer.screen)
	}
}

// CinematicOverlaySystem manages cinematic overlays.
type CinematicOverlaySystem struct{}

// NewCinematicOverlaySystem creates a new CinematicOverlaySystem.
func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	return &CinematicOverlaySystem{}
}

// Update updates all cinematic overlays in the world.
func (self *CinematicOverlaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Update(dt)
	}
}

// Draw renders all cinematic overlays to the screen.
func (self *CinematicOverlaySystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Draw(renderer.screen)
	}
}

// CooldownSystem manages cooldowns.
type CooldownSystem struct{}

// NewCooldownSystem creates a new CooldownSystem.
func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

// Update advances all cooldown managers in the world by the given delta time.
func (self *CooldownSystem) Update(world *World, dt float64) {
	entities := world.Query(CTCooldown)
	for _, e := range entities {
		cmAny, _ := world.GetComponent(e, CTCooldown)
		cm := cmAny.(*managers.CooldownManager)
		cm.Update(dt)
	}
}

// DelaySystem manages delays.
type DelaySystem struct{}

// NewDelaySystem creates a new DelaySystem.
func NewDelaySystem() *DelaySystem {
	return &DelaySystem{}
}

// Update advances all delay managers in the world by the given delta time.
func (self *DelaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTDelayer)
	for _, e := range entities {
		delayAny, _ := world.GetComponent(e, CTDelayer)
		delay := delayAny.(*managers.DelayManager)
		delay.Update(dt)
	}
}

// TextRenderSystem renders text components.
type TextRenderSystem struct{}

// NewTextRenderSystem creates a new TextRenderSystem.
func NewTextRenderSystem() *TextRenderSystem {
	return &TextRenderSystem{}
}

// Draw renders all text components in the world using their transforms.
func (self *TextRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTText, CTTransform)
	for _, entity := range entities {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		txt, _ := world.GetComponent(entity, CTText)
		textComp := txt.(*TextComponent)
		renderer.Flush()
		textComp.updateCache()
		op := &text.DrawOptions{}
		op.GeoM = t.Transform.Matrix()
		if offsetFunc, ok := alignmentOffsets[textComp.Alignment]; ok {
			offsetX, offsetY := offsetFunc(textComp.cachedWidth, textComp.cachedHeight)
			op.GeoM.Translate(offsetX, offsetY)
		}
		op.ColorScale = utils.RGBAToColorScale(textComp.Color)
		text.Draw(renderer.screen, textComp.Caption, textComp.fontFace, op)
	}
}

// SpriteRenderSystem renders sprite components.
type SpriteRenderSystem struct {
	tm *TextureManager
}

// NewSpriteRenderSystem creates a new SpriteRenderSystem with the given texture manager.
func NewSpriteRenderSystem(tm *TextureManager) *SpriteRenderSystem {
	return &SpriteRenderSystem{tm: tm}
}

// Draw renders all sprites in the world, sorted by Z-index for draw order.
func (self *SpriteRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entitiesWithSprite := world.Query(CTSprite, CTTransform)
	type drawableEntity struct {
		Entity Entity
		Z      float64
	}
	drawables := make([]drawableEntity, 0, len(entitiesWithSprite))
	for _, entity := range entitiesWithSprite {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		drawables = append(drawables, drawableEntity{Entity: entity, Z: t.Z})
	}

	sort.Slice(drawables, func(i, j int) bool {
		return drawables[i].Z < drawables[j].Z
	})

	for _, drawable := range drawables {
		entity := drawable.Entity
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		imgW, imgH := img.Size()
		srcX, srcY, srcW, srcH := s.GetSourceRect(float32(imgW), float32(imgH))
		destW32, destH32 := s.GetDestSize(srcW, srcH)
		destW, destH := float64(destW32), float64(destH32)

		effColor := s.Color
		effColor.A = uint8(float32(s.Color.A) * s.Opacity)

		renderer.DrawQuad(
			t.Position(),
			t.Scale(),
			t.Offset(),
			t.Origin(),
			t.Rotation(),
			img,
			effColor,
			srcX, srcY, srcX+srcW, srcY+srcH,
			destW, destH,
		)
	}
}
