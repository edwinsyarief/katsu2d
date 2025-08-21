package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

type FoliageSystem struct{}

func (self *FoliageSystem) Update(world *World, delta float64) {
	foliageControllerEntities := world.Query(CTFoliageController)
	if len(foliageControllerEntities) == 0 {
		return
	}
	foliageControllerEntity := foliageControllerEntities[0]
	foliageControllerComp, ok := world.GetComponent(foliageControllerEntity, CTFoliageController)
	if !ok {
		return
	}
	controller := foliageControllerComp.(*FoliageControllerComponent)

	// Update wind state.
	controller.windTime += delta
	controller.windScroll = controller.windScroll.Add(ebimath.Vector{X: 0.6 * delta * 60, Y: 0.4 * delta * 60})

	// Process external forces.
	currentFrameForceSources := make([]ForceSource, len(controller.externalForceSources))
	copy(currentFrameForceSources, controller.externalForceSources)
	controller.externalForceSources = controller.externalForceSources[:0]

	// Get all foliage entities to apply global effects like Perlin noise.
	foliageEntities := world.Query(CTFoliage, CTTransform, CTSprite)

	// Apply forces and gusts to relevant foliage entities.
	// We'll iterate through all foliage once to apply all forces, including the global noise.
	for _, entity := range foliageEntities {
		foliageComp, _ := world.GetComponent(entity, CTFoliage)
		foliage := foliageComp.(*FoliageComponent)
		spriteComp, _ := world.GetComponent(entity, CTSprite)
		sprite := spriteComp.(*SpriteComponent)
		transformComp, _ := world.GetComponent(entity, CTTransform)
		transform := transformComp.(*TransformComponent)

		// First frame setup: store original vertices.
		if len(foliage.OriginalVertices) == 0 && len(sprite.Vertices) > 0 {
			foliage.OriginalVertices = make([]ebiten.Vertex, len(sprite.Vertices))
			copy(foliage.OriginalVertices, sprite.Vertices)
		}

		// Calculate total force at the base of the foliage.
		foliageBasePos := transform.Position().Sub(transform.Origin())
		totalForce := 0.0

		// Add wind force based on Perlin noise.
		x, y := int(foliageBasePos.X/16), int(foliageBasePos.Y/16) // Scale to noise map coordinates
		x = (x + int(controller.windScroll.X)) % controller.noiseMapSize
		y = (y + int(controller.windScroll.Y)) % controller.noiseMapSize
		if x < 0 {
			x += controller.noiseMapSize
		}
		if y < 0 {
			y += controller.noiseMapSize
		}

		windValue := controller.getWindForceAt(float64(x), float64(y)) / 65535.0

		directionalSwayBias := controller.windDirection.X * controller.windForce * windValue
		oscillationAmplitude := controller.windForce * windValue * 0.5
		oscillationSway := math.Sin(controller.windTime*controller.windSpeed+foliage.SwaySeed) * oscillationAmplitude
		totalForce += directionalSwayBias + oscillationSway

		// Apply sway to each vertex.
		for i, originalVertex := range foliage.OriginalVertices {
			// Calculate vertex position relative to the foliage's fixed point.
			vertexLocalPos := ebimath.V(float64(originalVertex.DstX), float64(originalVertex.DstY))
			fixedPointPos := ebimath.V(float64(sprite.DstW)*foliage.PivotPoint.X, float64(sprite.DstH)*foliage.PivotPoint.Y)

			// Calculate the distance from the vertex to the fixed point.
			distFromFixed := vertexLocalPos.DistanceTo(fixedPointPos)
			maxDist := fixedPointPos.DistanceTo(ebimath.V(0, 0)) + fixedPointPos.DistanceTo(ebimath.V(float64(sprite.DstW), float64(sprite.DstH))) // A rough upper bound
			normalizedDist := 0.0
			if maxDist > 0 {
				normalizedDist = distFromFixed / maxDist
			}

			// Linear interpolation for the movement radius.
			swayRadius := controller.MinRadius + (controller.MaxRadius-controller.MinRadius)*normalizedDist

			// Calculate sway offset.
			swayOffset := totalForce * swayRadius

			// Update the vertex's position.
			sprite.Vertices[i].DstX = float32(float64(originalVertex.DstX) + swayOffset)

			// Mark the sprite as dirty so the renderer knows to update its mesh.
			sprite.dirty = true
		}
	}
}
