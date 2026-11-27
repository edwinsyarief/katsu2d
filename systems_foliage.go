package katsu2d

import (
	"math"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// FoliageSystem updates mesh vertices of foliage entities based on world-space wind waves,
// external forces, strong wind gusts, and vertex shading from FoliageControllerComponent.
type FoliageSystem struct {
	ctrlFilter     *teishoku.Filter[FoliageControllerComponent]
	foliageFilter  *teishoku.Filter4[TransformComponent, SpriteComponent, MeshComponent, FoliageComponent]
	camHandle      teishoku.ResourceHandle[ActiveCamera]
	activeGustsBuf []activeGustFrameData
	localForcesBuf []ForceSource
	localGustsBuf  []activeGustFrameData
	initialized    bool
}

// NewFoliageSystem creates and returns a new FoliageSystem.
func NewFoliageSystem() *FoliageSystem {
	return &FoliageSystem{}
}

// Initialize sets up the ECS filters and camera resource handle required by FoliageSystem.
func (s *FoliageSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	res := w.Resources()
	if ok, _ := teishoku.HasResource[ActiveCamera](res); !ok {
		teishoku.AddResource(res, &ActiveCamera{})
	}
	s.camHandle = teishoku.UseResource[ActiveCamera](res)
	s.ctrlFilter = teishoku.NewFilter[FoliageControllerComponent](w)
	s.foliageFilter = teishoku.NewFilter4[TransformComponent, SpriteComponent, MeshComponent, FoliageComponent](w)
	s.initialized = true
}

type foliageWindParams struct {
	windTimeSpeed   float64
	rippleTime      float64
	secRippleTime   float64
	windDirX        float64
	windDirY        float64
	windForce       float64
	rippleStrength  float64
	shadingStrength float64
}

// Update advances the wind simulation and applies sway, ripple, interaction, and shading to foliage meshes.
func (s *FoliageSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		if w == nil {
			return
		}
		s.Initialize(w)
	}
	if dt < 0 {
		return
	}

	var controller *FoliageControllerComponent
	s.activeGustsBuf = s.activeGustsBuf[:0]
	s.ctrlFilter.EachChunk(func(ctrls []FoliageControllerComponent) {
		for i := range ctrls {
			c := &ctrls[i]
			c.WindTime += dt
			collectFoliageActiveGusts(c, dt, &s.activeGustsBuf)
			controller = c
		}
	})

	if controller == nil {
		return
	}
	if len(controller.noiseTable) < foliageNoiseSize*foliageNoiseSize {
		if controller.Noise == nil {
			return
		}
		controller.noiseTable = buildFoliageNoiseTable(controller.Noise)
	}

	windTimeSpeed := controller.WindTime * controller.WindSpeed
	params := foliageWindParams{
		windTimeSpeed:   windTimeSpeed,
		rippleTime:      windTimeSpeed * 2.0,
		secRippleTime:   windTimeSpeed * 0.5,
		windDirX:        controller.WindDirection.X,
		windDirY:        controller.WindDirection.Y,
		windForce:       controller.WindForce,
		rippleStrength:  controller.RippleStrength,
		shadingStrength: Clamp(controller.ShadingStrength, 0.0, 1.0),
	}

	cam := s.camHandle.Get()
	hasCamCull := cam != nil && cam.HasCamera
	cullPad := math.Abs(params.windForce) + 32.0
	hasAnyForces := len(controller.ExternalForceSources) > 0
	hasAnyGusts := len(s.activeGustsBuf) > 0

	s.foliageFilter.EachChunk(func(
		transforms []TransformComponent,
		sprites []SpriteComponent,
		meshes []MeshComponent,
		foliages []FoliageComponent,
	) {
		for i := range foliages {
			tr := &transforms[i]
			spr := &sprites[i]
			mesh := &meshes[i]
			f := &foliages[i]

			var entBounds Rectangle
			hasKnownBounds := spr.Width > 0 && spr.Height > 0
			if hasKnownBounds && (hasCamCull || hasAnyForces || hasAnyGusts) {
				entBounds = foliageWorldBounds(tr, 0, 0, float64(spr.Width), float64(spr.Height), cullPad)
				if hasCamCull && !cam.IsVisible(entBounds) {
					continue
				}
			}

			wasDirty := mesh.IsDirty
			GenerateMesh(mesh, spr)

			vertCount := len(mesh.BaseVertices)
			if vertCount == 0 {
				continue
			}

			pivot := f.Pivot
			if pivot.X == 0 && pivot.Y == 0 {
				pivot = V(0.5, 1.0)
			}

			if wasDirty || f.cachedVertCount != vertCount || f.cachedHeight == 0 || f.cachedPivot != pivot {
				minX, maxX, minY, maxY := getVertexBounds(mesh.BaseVertices)
				f.cachedMinX = minX
				f.cachedMinY = minY
				f.cachedWidth = float64(maxX - minX)
				f.cachedHeight = float64(maxY - minY)
				f.cachedVertCount = vertCount
				f.cachedPivot = pivot
				precomputeFoliageWeights(f, mesh.BaseVertices, pivot)
			}
			if f.cachedHeight <= 0 {
				continue
			}

			if !hasKnownBounds && (hasCamCull || hasAnyForces || hasAnyGusts) {
				entBounds = foliageWorldBounds(
					tr,
					float64(f.cachedMinX), float64(f.cachedMinY),
					f.cachedWidth, f.cachedHeight,
					cullPad,
				)
				if hasCamCull && !cam.IsVisible(entBounds) {
					continue
				}
			}

			if len(mesh.Vertices) != vertCount {
				if cap(mesh.Vertices) < vertCount {
					mesh.Vertices = make([]ebiten.Vertex, vertCount)
				} else {
					mesh.Vertices = mesh.Vertices[:vertCount]
				}
				copy(mesh.Vertices, mesh.BaseVertices)
			}

			s.localForcesBuf = s.localForcesBuf[:0]
			if hasAnyForces {
				for j := range controller.ExternalForceSources {
					fs := controller.ExternalForceSources[j]
					if fs.Radius > 0 && entBounds.IntersectsCircle(fs.Position, fs.Radius) {
						s.localForcesBuf = append(s.localForcesBuf, fs)
					}
				}
			}

			s.localGustsBuf = s.localGustsBuf[:0]
			if hasAnyGusts {
				for j := range s.activeGustsBuf {
					g := s.activeGustsBuf[j]
					maxSpan := math.Max(g.gust.Length, g.gust.Width) * 0.5
					if maxSpan > 0 && entBounds.IntersectsCircle(g.pos, maxSpan) {
						s.localGustsBuf = append(s.localGustsBuf, g)
					}
				}
			}

			applyFoliageWind(tr, mesh, f, pivot, &params, controller, s.localForcesBuf, s.localGustsBuf)
		}
	})
}

func collectFoliageActiveGusts(controller *FoliageControllerComponent, dt float64, out *[]activeGustFrameData) {
	n := 0
	gusts := controller.StrongWindGusts
	for i := range gusts {
		gust := &gusts[i]
		if !gust.Active {
			continue
		}

		gust.ElapsedTime += dt
		if gust.Duration <= 0 || gust.ElapsedTime >= gust.Duration {
			gust.Active = false
			continue
		}

		strengthMultiplier := 1.0
		if gust.FadeInDuration > 0 && gust.ElapsedTime < gust.FadeInDuration {
			strengthMultiplier = gust.ElapsedTime / gust.FadeInDuration
		} else if gust.FadeOutDuration > 0 && gust.ElapsedTime > gust.Duration-gust.FadeOutDuration {
			strengthMultiplier = (gust.Duration - gust.ElapsedTime) / gust.FadeOutDuration
		}
		strengthMultiplier = Clamp(strengthMultiplier, 0.0, 1.0)

		progress := gust.ElapsedTime / gust.Duration
		currentPos := gust.StartPos.Add(gust.EndPos.Sub(gust.StartPos).ScaleF(progress))

		*out = append(*out, activeGustFrameData{
			gust:               *gust,
			pos:                currentPos,
			strengthMultiplier: strengthMultiplier,
		})

		gusts[n] = *gust
		n++
	}
	controller.StrongWindGusts = gusts[:n]
}

func foliageWorldBounds(tr *TransformComponent, localMinX, localMinY, width, height, pad float64) Rectangle {
	sx := math.Abs(tr.Scale.X)
	sy := math.Abs(tr.Scale.Y)
	if sx == 0 {
		sx = 1
	}
	if sy == 0 {
		sy = 1
	}
	w := width * sx
	h := height * sy
	if tr.Rotation != 0 {
		r := math.Hypot(w, h) + pad
		cx := tr.Position.X - tr.Origin.X
		cy := tr.Position.Y - tr.Origin.Y
		return Rectangle{
			Min: Vector{X: cx - r, Y: cy - r},
			Max: Vector{X: cx + r, Y: cy + r},
		}
	}
	minX := tr.Position.X - tr.Origin.X + (localMinX-tr.Offset.X)*sx
	minY := tr.Position.Y - tr.Origin.Y + (localMinY-tr.Offset.Y)*sy
	return Rectangle{
		Min: Vector{X: minX - pad, Y: minY - pad},
		Max: Vector{X: minX + w + pad, Y: minY + h + pad},
	}
}

func precomputeFoliageWeights(f *FoliageComponent, baseVerts []ebiten.Vertex, pivot Vector) {
	vertCount := len(baseVerts)
	if cap(f.cachedWeights) < vertCount {
		f.cachedWeights = make([]float64, vertCount)
	} else {
		f.cachedWeights = f.cachedWeights[:vertCount]
	}

	minX := float64(f.cachedMinX)
	minY := float64(f.cachedMinY)
	invHeight := 1.0 / f.cachedHeight
	var invWidth float64
	if f.cachedWidth > 0 {
		invWidth = 1.0 / f.cachedWidth
	}

	edgeY := (pivot.Y >= 0.999 || pivot.Y <= 0.001) && pivot.X > 0.001 && pivot.X < 0.999
	edgeX := (pivot.X >= 0.999 || pivot.X <= 0.001) && pivot.Y > 0.001 && pivot.Y < 0.999

	for v := range vertCount {
		ny := (float64(baseVerts[v].DstY) - minY) * invHeight
		nx := 0.5
		if invWidth > 0 {
			nx = (float64(baseVerts[v].DstX) - minX) * invWidth
		}
		dx := math.Abs(nx - pivot.X)
		dy := math.Abs(ny - pivot.Y)

		var weight float64
		switch {
		case edgeY:
			if dy > 0 {
				weight = dy * math.Sqrt(dy) * (0.85 + 0.30*dx)
			}
		case edgeX:
			if dx > 0 {
				weight = dx * math.Sqrt(dx) * (0.85 + 0.30*dy)
			}
		default:
			dist := math.Hypot(dx, dy)
			weight = dist * math.Sqrt(dist)
		}
		f.cachedWeights[v] = weight
	}
}

func applyFoliageWind(
	tr *TransformComponent,
	mesh *MeshComponent,
	f *FoliageComponent,
	pivot Vector,
	p *foliageWindParams,
	ctrl *FoliageControllerComponent,
	forces []ForceSource,
	gusts []activeGustFrameData,
) {
	// Hoist per-foliage sway terms outside the per-vertex loop.
	swayFreq := 1.0 + ctrl.sampleNoise(f.SwaySeed*3.7, 0)*0.2
	swayPhase := p.windTimeSpeed*0.1*swayFreq + f.SwaySeed
	swayNoise := ctrl.sampleNoise(swayPhase*4.0, f.SwaySeed*2.3)
	swayValue := math.Sin(swayPhase + swayNoise*10.0)
	baseDisplacement := (swayValue*0.45 + 0.55) * p.windForce

	minY := f.cachedMinY
	invHeight64 := 1.0 / f.cachedHeight
	invHeight32 := float32(invHeight64)
	posX := tr.Position.X
	posY := tr.Position.Y

	baseVerts := mesh.BaseVertices
	verts := mesh.Vertices
	weights := f.cachedWeights
	vertCount := len(baseVerts)

	hasForces := len(forces) > 0
	hasGusts := len(gusts) > 0
	shading := p.shadingStrength
	invWindNorm := 1.0 / math.Max(1.0, math.Abs(p.windForce))

	for v := range vertCount {
		bx := baseVerts[v].DstX
		by := baseVerts[v].DstY
		swayFactor := weights[v]

		if swayFactor == 0 {
			verts[v].DstX = bx
			verts[v].DstY = by
			if shading > 0 {
				ao := float32(1.0 - shading*0.85)
				verts[v].ColorR = ao
				verts[v].ColorG = ao
				verts[v].ColorB = ao
				verts[v].ColorA = 1
			}
			continue
		}

		wx := posX + float64(bx)
		wy := posY + float64(by)

		dotProduct := p.windDirX*wx + p.windDirY*wy
		primaryRipple := math.Sin(p.rippleTime - dotProduct*0.05)
		secondaryRipple := ctrl.sampleNoise(
			wx*0.15-p.windDirX*p.secRippleTime*6.0,
			wy*0.15-p.windDirY*p.secRippleTime*6.0,
		)
		rippleNoise := primaryRipple*0.7 + secondaryRipple*0.3

		displacement := baseDisplacement + rippleNoise*p.rippleStrength
		dispX := calculateDisplacement(displacement, swayFactor, p.windDirX)
		dispY := calculateDisplacement(displacement, swayFactor, p.windDirY)

		if hasForces {
			for fi := range forces {
				fs := forces[fi]
				dx := wx - fs.Position.X
				dy := wy - fs.Position.Y
				distSq := dx*dx + dy*dy
				radiusSq := fs.Radius * fs.Radius
				if distSq < radiusSq {
					dist := math.Sqrt(distSq)
					smoothFalloff := 1.0 - dist/fs.Radius
					falloff := smoothFalloff * smoothFalloff * smoothFalloff
					var dirX, dirY float64
					if dist > 0.0001 {
						invDist := 1.0 / dist
						dirX = dx * invDist
						dirY = dy * invDist
					} else {
						dirY = -1.0
					}
					pushMag := fs.Strength * falloff * 3.0 * swayFactor
					dispX += dirX * pushMag
					dispY += dirY * pushMag
				}
			}
		}

		if hasGusts {
			for gi := range gusts {
				gd := &gusts[gi]
				gust := &gd.gust
				halfLength := gust.Length * 0.5
				halfWidth := gust.Width * 0.5
				if halfLength <= 0 || halfWidth <= 0 {
					continue
				}
				toGustX := wx - gd.pos.X
				toGustY := wy - gd.pos.Y
				signedAlong := toGustX*gust.Direction.X + toGustY*gust.Direction.Y
				distAlong := math.Abs(signedAlong)
				perpDist := math.Abs(toGustX*gust.Direction.Y - toGustY*gust.Direction.X)
				if perpDist >= halfWidth || distAlong >= halfLength {
					continue
				}
				widthSmooth := 1.0 - perpDist/halfWidth
				t := (halfLength - signedAlong) / (2.0 * halfLength)
				var lengthSmooth float64
				if t < 0.3 {
					u := t / 0.3
					lengthSmooth = u * u * (3.0 - 2.0*u)
				} else {
					u := (1.0 - t) / 0.7
					lengthSmooth = u * u
				}
				gustMag := gust.Strength * gd.strengthMultiplier * (widthSmooth * widthSmooth) * lengthSmooth * 2.5 * swayFactor
				dispX += gust.Direction.X * gustMag
				dispY += gust.Direction.Y * gustMag
			}
		}

		normalizedY := (by - minY) * invHeight32
		arcY := (dispX * dispX) * 0.12 * invHeight64
		if float64(normalizedY) > pivot.Y {
			arcY = -arcY
		}

		verts[v].DstX = bx + float32(dispX)
		verts[v].DstY = by + float32(dispY+arcY)

		if shading > 0 {
			ao := Lerp(1.0-shading*0.85, 1.0, Clamp(swayFactor, 0.0, 1.0))
			crest := Clamp(displacement*invWindNorm-0.5, -0.5, 0.5)
			crestBoost := crest * shading * 0.45
			verts[v].ColorR = float32(Clamp(ao+crestBoost*1.05, 0.35, 1.0))
			verts[v].ColorG = float32(Clamp(ao+crestBoost*0.95, 0.35, 1.0))
			verts[v].ColorB = float32(Clamp(ao+crestBoost*0.55, 0.35, 1.0))
			verts[v].ColorA = 1
		}
	}
}

func getVertexBounds(vertices []ebiten.Vertex) (minX, maxX, minY, maxY float32) {
	minX = vertices[0].DstX
	maxX = vertices[0].DstX
	minY = vertices[0].DstY
	maxY = vertices[0].DstY
	for _, v := range vertices[1:] {
		if v.DstX < minX {
			minX = v.DstX
		}
		if v.DstX > maxX {
			maxX = v.DstX
		}
		if v.DstY < minY {
			minY = v.DstY
		}
		if v.DstY > maxY {
			maxY = v.DstY
		}
	}
	return minX, maxX, minY, maxY
}

func calculateDisplacement(displacement, swayFactor, direction float64) float64 {
	return displacement * swayFactor * direction
}
