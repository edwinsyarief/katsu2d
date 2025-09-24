package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/lazyecs"
)

// GrassControllerSystem is responsible for simulating the physics of all grass entities in the world.
// It handles wind effects, external forces (like a player walking through), and the resulting sway of each grass blade.
type GrassControllerSystem struct{}

// NewGrassControllerSystem creates a new instance of the grass simulation system.
func NewGrassControllerSystem() *GrassControllerSystem {
	return &GrassControllerSystem{}
}

// Update is the main simulation loop for the grass system. It is called once per frame.
// It calculates and applies all forces to the grass, updating their sway and rotation.
func (self *GrassControllerSystem) Update(world *lazyecs.World, dt float64) {
	var cameraExist = false
	var cameraArea ebimath.Rectangle

	query1 := world.Query(CTCamera)
	for query1.Next() {
		cameraExist = true
		cameras, _ := lazyecs.GetComponentSlice[CameraComponent](query1)
		cameraArea = cameras[0].Area()
	}

	// Retrieve the single GrassController component, which holds global settings for the simulation.
	query2 := world.Query(CTGrassController)
	var controller *GrassControllerComponent = nil
	var currentFrameForceSources = make([]ForceSource, 0)
	var activeGustsForFrame = []activeGustFrameData{}
	for query2.Next() {
		for _, entity := range query2.Entities() {
			controller, _ = lazyecs.GetComponent[GrassControllerComponent](world, entity)
			if cameraExist {
				controller.RenderArea = cameraArea
			}

			// Advance the global wind timer and scroll values, used for procedural wind patterns.
			controller.WindTime += dt
			controller.WindScroll = controller.WindScroll.Add(ebimath.Vector{X: 0.6 * dt * 60, Y: 0.4 * dt * 60})

			// Make a shallow copy of external force sources for this frame to ensure a consistent state during calculations.
			currentFrameForceSources = make([]ForceSource, len(controller.ExternalForceSources))
			copy(currentFrameForceSources, controller.ExternalForceSources)

			// Process active strong wind gusts.
			newGusts := []*StrongWindGust{} // A new slice to hold gusts that are still active.
			for _, gust := range controller.StrongWindGusts {
				if !gust.Active {
					continue
				}

				// Update the gust's lifetime and deactivate it if it has expired.
				gust.ElapsedTime += dt
				if gust.ElapsedTime >= gust.Duration {
					gust.Active = false
					continue
				}

				// Calculate the gust's current strength, applying fade-in and fade-out effects.
				currentStrengthMultiplier := 1.0
				if gust.ElapsedTime < gust.FadeInDuration {
					// Fading in at the beginning of the gust's life.
					currentStrengthMultiplier = gust.ElapsedTime / gust.FadeInDuration
				} else if gust.ElapsedTime > gust.Duration-gust.FadeOutDuration {
					// Fading out at the end.
					currentStrengthMultiplier = (gust.Duration - gust.ElapsedTime) / gust.FadeOutDuration
				}
				currentStrengthMultiplier = math.Max(0, math.Min(1, currentStrengthMultiplier)) // Clamp between 0 and 1.

				// Determine the gust's current position as it travels from its start to end point.
				progress := gust.ElapsedTime / gust.Duration
				currentGustPos := gust.StartPos.Add(gust.EndPos.Sub(gust.StartPos).ScaleF(progress))

				activeGustsForFrame = append(activeGustsForFrame, activeGustFrameData{
					gust:               gust,
					pos:                currentGustPos,
					strengthMultiplier: currentStrengthMultiplier,
				})
				newGusts = append(newGusts, gust) // Keep the gust for the next frame.
			}
			controller.StrongWindGusts = newGusts // Replace the old slice with the filtered one.
		}
	}

	if controller == nil {
		return
	}

	// Before applying new forces, reset the accumulated force for all grass entities.
	// This ensures that forces from the previous frame don't carry over.
	query3 := world.Query(CTGrass, CTTransform)
	grasses := make(map[lazyecs.Entity]*GrassComponent)
	transforms := make(map[lazyecs.Entity]*TransformComponent)
	grassEntities := make([]lazyecs.Entity, 0)
	for query3.Next() {
		for _, entity := range query3.Entities() {
			grassEntities = append(grassEntities, entity)

			grass, _ := lazyecs.GetComponent[GrassComponent](world, entity)
			grass.AccumulatedForce = 0.0
			grasses[entity] = grass

			transform, _ := lazyecs.GetComponent[TransformComponent](world, entity)
			transforms[entity] = transform
		}
	}

	// --- APPLY FORCES FROM EXTERNAL SOURCES (e.g., player movement) ---
	for _, fs := range currentFrameForceSources {
		// Use the quadtree to efficiently find grass entities within the force source's radius.
		affectedObjects := controller.Quadtree.QueryCircle(fs.Position, fs.Radius)
		for _, entity := range affectedObjects {
			grass := grasses[entity]
			transform := transforms[entity]

			pos := transform.Position()
			dx := pos.X - fs.Position.X
			dy := pos.Y - fs.Position.Y
			distSq := dx*dx + dy*dy

			// Check if the grass is within the circular area of effect.
			if distSq < fs.Radius*fs.Radius {
				dist := math.Sqrt(distSq)

				// Use a cubic falloff function (1-t)^3 to ensure the force smoothly approaches zero
				// at the edge of the radius. This prevents sudden changes in force and reduces jiggling.
				t := dist / fs.Radius
				smoothFalloff := 1.0 - t
				falloff := smoothFalloff * smoothFalloff * smoothFalloff

				// Determine the direction of the force (pushing away from the center).
				var direction float64
				if dist > 0.0001 { // Avoid division by zero at the exact center.
					direction = dx / dist
				}

				// Calculate and apply the force, scaled by strength, falloff, and a base acceleration.
				forceAccel := direction * fs.Strength * falloff * controller.ForceBaseAcceleration * 2.5
				grass.AccumulatedForce += forceAccel
			}
		}
	}

	// --- APPLY FORCES FROM STRONG WIND GUSTS ---
	for _, gustData := range activeGustsForFrame {
		gust := gustData.gust
		currentGustPos := gustData.pos
		currentStrengthMultiplier := gustData.strengthMultiplier

		// Calculate the bounding box of the oriented rectangle representing the gust's area of effect.
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

		// Use the quadtree to efficiently find grass entities within the gust's bounding box.
		affectedObjects := controller.Quadtree.Query(gustRect)
		for _, entity := range affectedObjects {
			grass := grasses[entity]
			transform := transforms[entity]

			// Determine the grass blade's position relative to the gust's center and direction.
			grassToGust := transform.Position().Sub(currentGustPos)
			distAlong := grassToGust.Dot(gust.Direction)
			perpDist := math.Abs(grassToGust.X*gust.Direction.Y - grassToGust.Y*gust.Direction.X)

			// To prevent unnatural jiggling as gusts pass over, we remodel the gust force
			// to have a smooth falloff from the center, both along its length and width.
			var widthFalloff, lengthFalloff float64

			// Calculate a smooth quadratic falloff from the gust's centerline to its sides.
			if perpDist < halfWidth {
				t := perpDist / halfWidth
				smooth := 1.0 - t
				widthFalloff = smooth * smooth
			}

			// Calculate a smooth quadratic falloff from the gust's center to its front and back.
			if math.Abs(distAlong) < halfLength {
				t := math.Abs(distAlong) / halfLength
				smooth := 1.0 - t
				lengthFalloff = smooth * smooth
			}

			// Combine the falloffs and apply the force if the grass is affected.
			combinedFalloff := widthFalloff * lengthFalloff
			if combinedFalloff > 0 {
				forceDirectionX := gust.Direction.X
				forceAccel := forceDirectionX * gust.Strength * currentStrengthMultiplier * combinedFalloff * controller.ForceBaseAcceleration * 2.5
				grass.AccumulatedForce += forceAccel
			}
		}
	}

	// --- UPDATE GRASS PHYSICS (SPRING-DAMPER MODEL) ---
	for _, entity := range grassEntities {
		grass := grasses[entity]

		// A spring force constantly tries to pull the grass back to its upright position (0 sway).
		springForce := (0 - grass.InteractionSway) * controller.SwaySpringStrength
		totalForce := grass.AccumulatedForce + springForce

		// Apply the total force to the grass's velocity, then apply damping to slow it down over time.
		grass.SwayVelocity += totalForce * dt
		grass.SwayVelocity *= controller.SwayDamping
		grass.InteractionSway += grass.SwayVelocity * dt

		// A positional decay is re-introduced to ensure grass always returns to its upright state.
		// A pure spring-damper system could allow grass to get "stuck" in a swayed position
		// under certain conditions. This provides a necessary corrective pull.
		grass.InteractionSway *= math.Pow(0.85, dt*60)

		// To prevent tiny, rapid oscillations when the grass is nearly still,
		// snap its sway and velocity to zero if they fall below a small threshold.
		if math.Abs(grass.InteractionSway) < 0.0001 {
			grass.InteractionSway = 0.0
			grass.SwayVelocity = 0.0
		}

		transform := transforms[entity]
		pos := transform.Position()

		// --- BLEND WIND AND INTERACTION SWAY ---
		localWindForceMagnitude := controller.getWindForceAt(pos.X, pos.Y)

		// Check for any significant, immediate force being applied to the grass.
		forcePresent := math.Abs(grass.AccumulatedForce) > 0.001

		var directionalSwayBias float64
		var oscillationAmplitude float64
		var oscillationSway float64

		// Only apply wind effects if there's no significant external force.
		// This makes the grass react to interactions rather than being blown by the wind.
		if !forcePresent {
			// A constant directional push from the wind.
			directionalSwayBias = controller.WindDirection.X * controller.WindForce * localWindForceMagnitude
			// A gentle, oscillating sway to simulate rustling.
			oscillationAmplitude = controller.WindForce * localWindForceMagnitude * 0.5
			oscillationSway = math.Sin(controller.WindTime*controller.WindSpeed+grass.SwaySeed) * oscillationAmplitude
		}

		// Combine the wind effects (if any) with the sway from interactions.
		totalSway := directionalSwayBias + oscillationSway + grass.InteractionSway

		// Clamp the total sway to a maximum value to prevent the grass from bending too far.
		maxSway := math.Pi / 2.2
		if totalSway > maxSway {
			totalSway = maxSway
		} else if totalSway < -maxSway {
			totalSway = -maxSway
		}

		// Smoothly interpolate from the current sway to the target sway for a more fluid animation.
		grass.CurrentSway = ebimath.Lerp(grass.CurrentSway, totalSway, 1-math.Pow(0.001, dt))
		transform.SetRotation(grass.CurrentSway)
	}
}
