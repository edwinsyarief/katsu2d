package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// GrassControllerSystem is an update system that simulates the grass physics.
type GrassControllerSystem struct{}

// Update simulates the grass physics, applying forces and wind effects.
func (self *GrassControllerSystem) Update(world *World, delta float64) {
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

	controller.windTime += delta
	controller.windScroll = controller.windScroll.Add(ebimath.Vector{X: 0.6 * delta * 60, Y: 0.4 * delta * 60})

	currentFrameForceSources := make([]ForceSource, len(controller.externalForceSources))
	copy(currentFrameForceSources, controller.externalForceSources)

	activeGustsForFrame := []activeGustFrameData{}
	newGusts := []*StrongWindGust{}
	for _, gust := range controller.strongWindGusts {
		if !gust.Active {
			continue
		}
		gust.ElapsedTime += delta
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

	grassEntities := world.Query(CTGrass, CTTransform)
	for _, entity := range grassEntities {
		grassComp, _ := world.GetComponent(entity, CTGrass)
		grass := grassComp.(*GrassComponent)
		grass.AccumulatedForce = 0.0
	}

	for _, fs := range currentFrameForceSources {
		affectedObjects := controller.quadtree.QueryCircle(fs.Position, fs.Radius)
		for _, obj := range affectedObjects {
			transform := obj.(*TransformComponent)
			grassEntities := world.Query(CTGrass, CTTransform)
			for _, entity := range grassEntities {
				t, _ := world.GetComponent(entity, CTTransform)
				if t == transform {
					grassComp, _ := world.GetComponent(entity, CTGrass)
					grass := grassComp.(*GrassComponent)
					pos := transform.Position()
					dx := pos.X - fs.Position.X
					dy := pos.Y - fs.Position.Y
					distSq := dx*dx + dy*dy
					if distSq < fs.Radius*fs.Radius {
						dist := math.Sqrt(distSq)
						falloff := 1.0 - (dist / fs.Radius)
						direction := dx / dist
						forceAccel := direction * fs.Strength * falloff * controller.forceBaseAcceleration
						grass.AccumulatedForce += forceAccel
					}
				}
			}
		}
	}

	for _, gustData := range activeGustsForFrame {
		gust := gustData.gust
		currentGustPos := gustData.pos
		currentStrengthMultiplier := gustData.strengthMultiplier

		perp := ebimath.Vector{X: -gust.Direction.Y, Y: gust.Direction.X}.Normalized()
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
		for _, obj := range affectedObjects {
			transform := obj.(*TransformComponent)
			grassEntities := world.Query(CTGrass, CTTransform)
			for _, entity := range grassEntities {
				t, _ := world.GetComponent(entity, CTTransform)
				if t == transform {
					grassComp, _ := world.GetComponent(entity, CTGrass)
					grass := grassComp.(*GrassComponent)
					grassToGust := transform.Position().Sub(currentGustPos)
					distAlong := grassToGust.Dot(gust.Direction)
					perpDist := math.Abs(grassToGust.X*gust.Direction.Y - grassToGust.Y*gust.Direction.X)
					widthFalloff := 1.0 - (perpDist / (gust.Width / 2.0))
					widthFalloff = math.Max(0, math.Min(1, widthFalloff))
					lengthFalloff := 0.0
					if distAlong <= halfLength && distAlong >= -halfLength {
						normalizedAlong := (distAlong + halfLength) / gust.Length
						lengthFalloff = normalizedAlong
					}
					lengthFalloff = math.Max(0, math.Min(1, lengthFalloff))
					combinedFalloff := widthFalloff * lengthFalloff
					if combinedFalloff > 0 {
						forceDirectionX := gust.Direction.X
						forceAccel := forceDirectionX * gust.Strength * currentStrengthMultiplier * combinedFalloff * controller.forceBaseAcceleration
						grass.AccumulatedForce += forceAccel
					}
				}
			}
		}
	}

	for _, entity := range grassEntities {
		grassComp, _ := world.GetComponent(entity, CTGrass)
		grass := grassComp.(*GrassComponent)
		springForce := (0 - grass.InteractionSway) * controller.swaySpringStrength
		totalForce := grass.AccumulatedForce + springForce
		grass.SwayVelocity += totalForce * delta
		grass.SwayVelocity *= controller.swayDamping
		grass.InteractionSway += grass.SwayVelocity * delta
		transformComp, _ := world.GetComponent(entity, CTTransform)
		transform := transformComp.(*TransformComponent)
		pos := transform.Position()
		localWindForceMagnitude := controller.getWindForceAt(pos.X, pos.Y)
		directionalSwayBias := controller.windDirection.X * controller.windForce * localWindForceMagnitude
		oscillationAmplitude := controller.windForce * localWindForceMagnitude * 0.5
		oscillationSway := math.Sin(controller.windTime*controller.windSpeed+grass.SwaySeed) * oscillationAmplitude
		totalSway := directionalSwayBias + oscillationSway + grass.InteractionSway
		maxSway := math.Pi / 2.2
		if totalSway > maxSway {
			totalSway = maxSway
		} else if totalSway < -maxSway {
			totalSway = -maxSway
		}
		transform.SetRotation(totalSway)
	}
}
