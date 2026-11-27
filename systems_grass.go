package katsu2d

import (
	"math"

	"github.com/edwinsyarief/teishoku"
)

type activeGustFrameData struct {
	gust               StrongWindGust
	pos                Vector
	strengthMultiplier float64
}

// GrassControllerSystem simulates ambient wind, external radial forces,
// strong wind gusts, and spring-damper recovery physics for all grass entities.
type GrassControllerSystem struct {
	ctrlFilter     *teishoku.Filter[GrassControllerComponent]
	grassFilter    *teishoku.Filter3[TransformComponent, SpriteComponent, GrassComponent]
	grassBuilder   *teishoku.Builder2[TransformComponent, GrassComponent]
	bladeBuilder   *teishoku.Builder3[TransformComponent, SpriteComponent, GrassComponent]
	camHandle      teishoku.ResourceHandle[ActiveCamera]
	controllersBuf []*GrassControllerComponent
	activeGustsBuf []activeGustFrameData
	affectedBuf    []teishoku.Entity
	initialized    bool
}

// NewGrassControllerSystem creates a new GrassControllerSystem.
func NewGrassControllerSystem() *GrassControllerSystem {
	return &GrassControllerSystem{}
}

// Initialize caches the ECS filters, builders, and camera handle required by the system.
func (s *GrassControllerSystem) Initialize(w *teishoku.World) {
	if s.initialized || w == nil {
		return
	}
	res := w.Resources()
	if ok, _ := teishoku.HasResource[ActiveCamera](res); !ok {
		teishoku.AddResource(res, &ActiveCamera{})
	}
	s.camHandle = teishoku.UseResource[ActiveCamera](res)
	s.ctrlFilter = teishoku.NewFilter[GrassControllerComponent](w)
	s.grassFilter = teishoku.NewFilter3[TransformComponent, SpriteComponent, GrassComponent](w)
	s.grassBuilder = teishoku.NewBuilder2[TransformComponent, GrassComponent](w)
	s.bladeBuilder = teishoku.NewBuilder3[TransformComponent, SpriteComponent, GrassComponent](w)
	s.initialized = true
}

// Update advances the grass wind and interaction physics by dt seconds.
func (s *GrassControllerSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		if w == nil {
			return
		}
		s.Initialize(w)
	}
	if dt < 0 {
		return
	}

	s.controllersBuf = s.controllersBuf[:0]
	ctrlQuery := s.ctrlFilter.Query()
	for ctrlQuery.Next() {
		s.controllersBuf = append(s.controllersBuf, ctrlQuery.Get())
	}
	if len(s.controllersBuf) == 0 {
		return
	}

	// Reset accumulated forces once before collecting all controllers' interactions.
	s.grassFilter.EachChunk(func(_ []TransformComponent, _ []SpriteComponent, grasses []GrassComponent) {
		for i := range grasses {
			grasses[i].AccumulatedForce = 0.0
		}
	})

	cam := s.camHandle.Get()
	hasCamCull := cam != nil && cam.HasCamera
	var camCullRect Rectangle
	if hasCamCull {
		const pad = 64.0
		camCullRect = Rectangle{
			Min: Vector{X: cam.ViewBounds.Min.X - pad, Y: cam.ViewBounds.Min.Y - pad},
			Max: Vector{X: cam.ViewBounds.Max.X + pad, Y: cam.ViewBounds.Max.Y + pad},
		}
	}

	decay := math.Pow(0.85, dt*60)
	lerpFactor := 1.0 - math.Pow(0.001, dt)
	multiCtrl := len(s.controllersBuf) > 1

	for _, controller := range s.controllersBuf {
		controller.WindTime += dt
		scrollScale := controller.WindSpeed * dt * 28.0
		controller.WindScroll.X += controller.WindDirection.X * scrollScale
		controller.WindScroll.Y += controller.WindDirection.Y * scrollScale

		s.collectActiveGusts(controller, dt)

		if controller.Quadtree != nil && controller.Quadtree.root != nil {
			s.applyExternalForceSources(controller)
			s.applyStrongWindGusts(controller)
		}

		// Precompute frame-invariant math outside the per-blade loop.
		effectiveWindDir := controller.WindDirection.X + controller.WindDirection.Y*0.35
		windBiasScale := effectiveWindDir * controller.WindForce
		windOscAmpScale := controller.WindForce * 0.5
		windPhase := controller.WindTime * controller.WindSpeed
		springStrength := controller.SwaySpringStrength
		swayDamping := math.Pow(math.Max(0, controller.SwayDamping), dt*60)
		shadingStrength := Clamp(controller.WindShadingStrength, 0.0, 1.0)
		hasRenderArea := !controller.RenderArea.IsEmpty()
		renderArea := controller.RenderArea

		if !multiCtrl {
			s.grassFilter.EachChunk(func(transforms []TransformComponent, sprites []SpriteComponent, grasses []GrassComponent) {
				for i := range grasses {
					tr := &transforms[i]
					px, py := tr.Position.X, tr.Position.Y
					if hasCamCull && (px < camCullRect.Min.X || px > camCullRect.Max.X || py < camCullRect.Min.Y || py > camCullRect.Max.Y) {
						continue
					}
					if hasRenderArea && (px < renderArea.Min.X || px >= renderArea.Max.X || py < renderArea.Min.Y || py >= renderArea.Max.Y) {
						continue
					}
					updateGrassBlade(
						tr, &sprites[i], &grasses[i], controller,
						dt, decay, lerpFactor, windBiasScale, windOscAmpScale,
						windPhase, springStrength, swayDamping, shadingStrength,
					)
				}
			})
		} else if controller.Quadtree != nil && controller.Quadtree.root != nil {
			queryRect := controller.Quadtree.Bounds()
			if hasCamCull {
				queryRect = intersectRects(queryRect, camCullRect)
			}
			if hasRenderArea {
				queryRect = intersectRects(queryRect, renderArea)
			}
			if !queryRect.IsEmpty() {
				s.affectedBuf = controller.Quadtree.QueryInto(queryRect, s.affectedBuf)
				for _, entity := range s.affectedBuf {
					tr, sp, g := s.bladeBuilder.Get(entity)
					if tr == nil || sp == nil || g == nil {
						continue
					}
					updateGrassBlade(
						tr, sp, g, controller,
						dt, decay, lerpFactor, windBiasScale, windOscAmpScale,
						windPhase, springStrength, swayDamping, shadingStrength,
					)
				}
			}
		}
	}

	clear(s.controllersBuf)
	s.controllersBuf = s.controllersBuf[:0]
}

func (s *GrassControllerSystem) collectActiveGusts(controller *GrassControllerComponent, dt float64) {
	s.activeGustsBuf = s.activeGustsBuf[:0]
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

		s.activeGustsBuf = append(s.activeGustsBuf, activeGustFrameData{
			gust:               *gust,
			pos:                currentPos,
			strengthMultiplier: strengthMultiplier,
		})

		gusts[n] = *gust
		n++
	}
	controller.StrongWindGusts = gusts[:n]
}

func updateGrassBlade(
	tr *TransformComponent,
	sp *SpriteComponent,
	g *GrassComponent,
	controller *GrassControllerComponent,
	dt, decay, lerpFactor float64,
	windBiasScale, windOscAmpScale, windPhase float64,
	springStrength, swayDamping, shadingStrength float64,
) {
	const maxSway = math.Pi / 2.2
	const invMaxSway = 1.0 / maxSway

	springForce := -g.InteractionSway * springStrength
	totalForce := g.AccumulatedForce + springForce

	g.SwayVelocity += totalForce * dt
	g.SwayVelocity *= swayDamping
	g.InteractionSway += g.SwayVelocity * dt
	g.InteractionSway *= decay

	if math.Abs(g.InteractionSway) < 0.0001 {
		g.InteractionSway = 0.0
		g.SwayVelocity = 0.0
	}

	ambientWeight := Clamp(1.0-math.Abs(g.AccumulatedForce)*0.02, 0.15, 1.0)
	localWindForce := controller.getWindForceAt(tr.Position.X, tr.Position.Y)
	directionalSwayBias := windBiasScale * localWindForce * ambientWeight
	oscillationSway := math.Sin(windPhase+g.SwaySeed) * (windOscAmpScale * localWindForce) * ambientWeight

	totalSway := Clamp(directionalSwayBias+oscillationSway+g.InteractionSway, -maxSway, maxSway)
	g.CurrentSway = Lerp(g.CurrentSway, totalSway, lerpFactor)
	tr.Rotation = g.CurrentSway

	if shadingStrength > 0 {
		swayNorm := math.Abs(g.CurrentSway) * invMaxSway
		crest := Clamp(localWindForce*0.55+swayNorm*0.45-0.35, -1.0, 1.0)
		mid := 1.0 - shadingStrength*0.3
		rMul := Clamp(mid+crest*shadingStrength*0.95, 0.3, 1.0)
		gMul := Clamp(mid+crest*shadingStrength*0.85, 0.3, 1.0)
		bMul := Clamp(mid+crest*shadingStrength*0.45-math.Max(0, crest)*shadingStrength*0.35, 0.3, 1.0)
		sp.Color.R = uint8(rMul * 255.0)
		sp.Color.G = uint8(gMul * 255.0)
		sp.Color.B = uint8(bMul * 255.0)
	}
}

func intersectRects(a, b Rectangle) Rectangle {
	return Rectangle{
		Min: Vector{X: math.Max(a.Min.X, b.Min.X), Y: math.Max(a.Min.Y, b.Min.Y)},
		Max: Vector{X: math.Min(a.Max.X, b.Max.X), Y: math.Min(a.Max.Y, b.Max.Y)},
	}
}

func tangentialPushDirection(dx, dy, swaySeed float64) float64 {
	seedSign := 1.0
	if math.Sin(swaySeed) < 0 {
		seedSign = -1.0
	}
	tangential := seedSign*0.65 + Clamp(dy, -1.0, 1.0)*0.35
	blend := Clamp(1.0-math.Abs(dx)*2.0, 0.0, 1.0)
	return dx*(1.0-blend) + tangential*blend
}

func (s *GrassControllerSystem) applyExternalForceSources(controller *GrassControllerComponent) {
	baseForceScale := controller.ForceBaseAcceleration * 2.5
	for i := range controller.ExternalForceSources {
		fs := controller.ExternalForceSources[i]
		if fs.Radius <= 0 {
			continue
		}

		radiusSq := fs.Radius * fs.Radius
		s.affectedBuf = controller.Quadtree.QueryCircleInto(fs.Position, fs.Radius, s.affectedBuf)

		for _, entity := range s.affectedBuf {
			transform, grass := s.grassBuilder.Get(entity)
			if transform == nil || grass == nil {
				continue
			}

			dx := transform.Position.X - fs.Position.X
			dy := transform.Position.Y - fs.Position.Y
			distSq := dx*dx + dy*dy

			if distSq < radiusSq {
				dist := math.Sqrt(distSq)
				smoothFalloff := 1.0 - dist/fs.Radius
				falloff := smoothFalloff * smoothFalloff * smoothFalloff

				var dirX, dirY float64
				if dist > 0.0001 {
					invDist := 1.0 / dist
					dirX = dx * invDist
					dirY = dy * invDist
				}

				direction := tangentialPushDirection(dirX, dirY, grass.SwaySeed)
				grass.AccumulatedForce += direction * fs.Strength * falloff * baseForceScale
			}
		}
	}
}

func (s *GrassControllerSystem) applyStrongWindGusts(controller *GrassControllerComponent) {
	baseForceScale := controller.ForceBaseAcceleration * 2.5
	for i := range s.activeGustsBuf {
		gustData := &s.activeGustsBuf[i]
		gust := &gustData.gust
		halfLength := gust.Length * 0.5
		halfWidth := gust.Width * 0.5
		if halfLength <= 0 || halfWidth <= 0 {
			continue
		}

		currentGustPos := gustData.pos
		perp := Vector{X: -gust.Direction.Y, Y: gust.Direction.X}.Normalize()
		alongHalf := gust.Direction.ScaleF(halfLength)
		perpHalf := perp.ScaleF(halfWidth)

		c1 := currentGustPos.Add(alongHalf).Add(perpHalf)
		c2 := currentGustPos.Add(alongHalf).Sub(perpHalf)
		c3 := currentGustPos.Sub(alongHalf).Add(perpHalf)
		c4 := currentGustPos.Sub(alongHalf).Sub(perpHalf)

		gustRect := Rectangle{
			Min: Vector{
				X: min(c1.X, c2.X, c3.X, c4.X),
				Y: min(c1.Y, c2.Y, c3.Y, c4.Y),
			},
			Max: Vector{
				X: max(c1.X, c2.X, c3.X, c4.X),
				Y: max(c1.Y, c2.Y, c3.Y, c4.Y),
			},
		}

		gustForceBase := gust.Strength * gustData.strengthMultiplier * baseForceScale
		s.affectedBuf = controller.Quadtree.QueryInto(gustRect, s.affectedBuf)

		for _, entity := range s.affectedBuf {
			transform, grass := s.grassBuilder.Get(entity)
			if transform == nil || grass == nil {
				continue
			}

			grassToGust := Vector(transform.Position).Sub(currentGustPos)
			signedAlong := grassToGust.Dot(gust.Direction)
			distAlong := math.Abs(signedAlong)
			perpDist := math.Abs(grassToGust.X*gust.Direction.Y - grassToGust.Y*gust.Direction.X)

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

			combinedFalloff := (widthSmooth * widthSmooth) * lengthSmooth
			if combinedFalloff > 0 {
				pushDir := tangentialPushDirection(gust.Direction.X, gust.Direction.Y, grass.SwaySeed)
				grass.AccumulatedForce += pushDir * gustForceBase * combinedFalloff
			}
		}
	}
}
