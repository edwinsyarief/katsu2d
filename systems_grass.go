package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// GrassControllerSystem is an update system that simulates the grass physics.
type GrassControllerSystem struct{}

func NewGrassControllerSystem() *GrassControllerSystem {
	return &GrassControllerSystem{}
}

// Update simulates the grass physics, applying forces and wind effects.
func (self *GrassControllerSystem) Update(world *World, dt float64) {
	grassControllerEntities := world.Query(CTGrassController)
	if len(grassControllerEntities) == 0 {
		return
	}
	grassControllerEntity := grassControllerEntities[0]
	grassControllerComp, ok := world.GetComponent(grassControllerEntity, CTGrassController)
	if !ok {
		return
	}
	controller := grassControllerComp.(*GrassControllerComponent)

	cameraEntities := world.Query(CTCamera)
	if len(cameraEntities) > 0 {
		cameraEntity := cameraEntities[0]
		cameraComp, ok := world.GetComponent(cameraEntity, CTCamera)
		if ok {
			camera := cameraComp.(*CameraComponent)
			controller.renderArea = camera.Area()
		}
	}

	controller.windTime += dt
	controller.windScroll = controller.windScroll.Add(ebimath.Vector{X: 0.6 * dt * 60, Y: 0.4 * dt * 60})

	currentFrameForceSources := make([]ForceSource, len(controller.externalForceSources))
	copy(currentFrameForceSources, controller.externalForceSources)

	activeGustsForFrame := []activeGustFrameData{}
	newGusts := []*StrongWindGust{}
	for _, gust := range controller.strongWindGusts {
		if !gust.Active {
			continue
		}
		gust.ElapsedTime += dt
		if gust.ElapsedTime >= gust.Duration {
			gust.Active = false
			continue
		}
		currentStrengthMultiplier := 1.0
		if gust.ElapsedTime < gust.FadeInDuration {
			currentStrengthMultiplier = gust.ElapsedTime / gust.FadeInDuration
		} else if gust.ElapsedTime > gust.Duration-gust.FadeOutDuration {
			currentStrengthMultiplier = (gust.Duration - gust.ElapsedTime) / gust.FadeOutDuration
		}
		currentStrengthMultiplier = math.Max(0, math.Min(1, currentStrengthMultiplier))
		progress := gust.ElapsedTime / gust.Duration
		currentGustPos := gust.StartPos.Add(gust.EndPos.Sub(gust.StartPos).ScaleF(progress))
		activeGustsForFrame = append(activeGustsForFrame, activeGustFrameData{
			gust:               gust,
			pos:                currentGustPos,
			strengthMultiplier: currentStrengthMultiplier,
		})
		newGusts = append(newGusts, gust)
	}
	controller.strongWindGusts = newGusts

	// This part is crucial for resetting the accumulated force before the physics step.
	// We only need to reset the force for the entities that might be affected this frame.
	grassEntities := world.Query(CTGrass, CTTransform)
	for _, entity := range grassEntities {
		grassComp, _ := world.GetComponent(entity, CTGrass)
		grass := grassComp.(*GrassComponent)
		grass.AccumulatedForce = 0.0
	}

	// Apply forces from external sources
	for _, fs := range currentFrameForceSources {
		affectedObjects := controller.quadtree.QueryCircle(fs.Position, fs.Radius)
		for _, entity := range affectedObjects {
			// Corrected logic: Get components directly from the entity found by the quadtree.
			grassAny, ok := world.GetComponent(entity, CTGrass)
			if !ok {
				continue
			}
			grass := grassAny.(*GrassComponent)
			transformAny, ok := world.GetComponent(entity, CTTransform)
			if !ok {
				continue
			}
			transform := transformAny.(*TransformComponent)

			pos := transform.Position()
			dx := pos.X - fs.Position.X
			dy := pos.Y - fs.Position.Y
			distSq := dx*dx + dy*dy
			if distSq < fs.Radius*fs.Radius {
				dist := math.Sqrt(distSq)

				// =================================================================================
				// FIX: Use an even smoother (cubic) falloff to further reduce jiggling at the edges.
				// This ensures the force approaches zero very gently.
				t := dist / fs.Radius
				smoothFalloff := 1.0 - t
				falloff := smoothFalloff * smoothFalloff * smoothFalloff
				// =================================================================================

				direction := 0.0
				// Add a small tolerance check to avoid division by zero
				if dist > 0.0001 {
					direction = dx / dist
				}

				// Increase force to make grass bend faster
				forceAccel := direction * fs.Strength * falloff * controller.forceBaseAcceleration * 2.5
				grass.AccumulatedForce += forceAccel
			}
		}
	}

	// Apply forces from strong wind gusts
	for _, gustData := range activeGustsForFrame {
		gust := gustData.gust
		currentGustPos := gustData.pos
		currentStrengthMultiplier := gustData.strengthMultiplier

		perp := ebimath.Vector{X: -gust.Direction.Y, Y: gust.Direction.X}.Normalize()
		halfLength := gust.Length / 2.0
		halfWidth := gust.Width / 2.0
		corner1 := currentGustPos.Add(gust.Direction.ScaleF(halfLength)).Add(perp.ScaleF(halfWidth))
		corner2 := currentGustPos.Add(gust.Direction.ScaleF(halfLength)).Add(perp.ScaleF(-halfWidth))
		corner3 := currentGustPos.Add(gust.Direction.ScaleF(-halfLength)).Add(perp.ScaleF(halfWidth))
		corner4 := currentGustPos.Add(gust.Direction.ScaleF(-halfLength)).Add(perp.ScaleF(-halfWidth))
		minX := math.Min(corner1.X, math.Min(corner2.X, math.Min(corner3.X, corner4.X)))
		maxX := math.Max(corner1.X, math.Max(corner2.X, math.Max(corner3.X, corner4.X)))
		minY := math.Min(corner1.Y, math.Min(corner2.Y, math.Min(corner3.Y, corner4.Y)))
		maxY := math.Max(corner1.Y, math.Max(corner2.Y, math.Max(corner3.Y, corner4.Y)))
		gustRect := ebimath.Rectangle{
			Min: ebimath.Vector{X: minX, Y: minY},
			Max: ebimath.Vector{X: maxX, Y: maxY},
		}

		affectedObjects := controller.quadtree.Query(gustRect)
		for _, entity := range affectedObjects {
			// Corrected logic: Get components directly from the entity found by the quadtree.
			grassAny, ok := world.GetComponent(entity, CTGrass)
			if !ok {
				continue
			}
			grass := grassAny.(*GrassComponent)
			transformAny, ok := world.GetComponent(entity, CTTransform)
			if !ok {
				continue
			}
			transform := transformAny.(*TransformComponent)

			grassToGust := transform.Position().Sub(currentGustPos)
			distAlong := grassToGust.Dot(gust.Direction)
			perpDist := math.Abs(grassToGust.X*gust.Direction.Y - grassToGust.Y*gust.Direction.X)

			// =================================================================================
			// FIX: Remodel gust force to have a smooth falloff from the center,
			// preventing jiggling and unnatural movement as gusts pass over grass.
			var widthFalloff, lengthFalloff float64

			// Calculate smooth falloff from the gust's centerline to its sides.
			if perpDist < halfWidth {
				t := perpDist / halfWidth
				smooth := 1.0 - t
				widthFalloff = smooth * smooth
			}

			// Calculate smooth falloff from the gust's center to its front and back.
			if math.Abs(distAlong) < halfLength {
				t := math.Abs(distAlong) / halfLength
				smooth := 1.0 - t
				lengthFalloff = smooth * smooth
			}
			// =================================================================================

			combinedFalloff := widthFalloff * lengthFalloff
			if combinedFalloff > 0 {
				forceDirectionX := gust.Direction.X
				// Increase force to make grass bend faster
				forceAccel := forceDirectionX * gust.Strength * currentStrengthMultiplier * combinedFalloff * controller.forceBaseAcceleration * 2.5
				grass.AccumulatedForce += forceAccel
			}
		}
	}

	// Update grass physics based on accumulated forces
	for _, entity := range grassEntities {
		grassComp, _ := world.GetComponent(entity, CTGrass)
		grass := grassComp.(*GrassComponent)
		springForce := (0 - grass.InteractionSway) * controller.swaySpringStrength
		totalForce := grass.AccumulatedForce + springForce
		grass.SwayVelocity += totalForce * dt
		grass.SwayVelocity *= controller.swayDamping
		grass.InteractionSway += grass.SwayVelocity * dt

		// =================================================================================
		// FIX: Re-introduced positional decay to ensure the grass always returns to its
		// upright state. The pure spring-damper system was allowing the grass to get
		// "stuck", and this provides a necessary corrective force.
		grass.InteractionSway *= math.Pow(0.85, dt*60)
		// =================================================================================

		// This part fixes the fast waving at the edges. When the sway becomes very small,
		// it snaps to zero to prevent tiny, rapid oscillations.
		if math.Abs(grass.InteractionSway) < 0.0001 {
			grass.InteractionSway = 0.0
			grass.SwayVelocity = 0.0
		}

		transformComp, _ := world.GetComponent(entity, CTTransform)
		transform := transformComp.(*TransformComponent)
		pos := transform.Position()

		// Logic to smoothly blend wind and force-based sway.
		// We use the persistent InteractionSway to determine how much a force is already bending the grass.
		localWindForceMagnitude := controller.getWindForceAt(pos.X, pos.Y)

		// A more direct check for a significant, immediate force to disable wind animation
		forcePresent := math.Abs(grass.AccumulatedForce) > 0.001

		var directionalSwayBias float64
		var oscillationAmplitude float64
		var oscillationSway float64

		if !forcePresent {
			// Apply wind effects only if there is no significant external force
			directionalSwayBias = controller.windDirection.X * controller.windForce * localWindForceMagnitude
			oscillationAmplitude = controller.windForce * localWindForceMagnitude * 0.5
			oscillationSway = math.Sin(controller.windTime*controller.windSpeed+grass.SwaySeed) * oscillationAmplitude
		}

		totalSway := directionalSwayBias + oscillationSway + grass.InteractionSway
		maxSway := math.Pi / 2.2
		if totalSway > maxSway {
			totalSway = maxSway
		} else if totalSway < -maxSway {
			totalSway = -maxSway
		}
		grass.CurrentSway = ebimath.Lerp(grass.CurrentSway, totalSway, 1-math.Pow(0.001, dt))
		transform.SetRotation(grass.CurrentSway)
	}
}
