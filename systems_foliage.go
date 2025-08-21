package katsu2d

import (
	"math"
)

// FoliageSystem is an update system that simulates foliage physics.
// It assumes that there is only one FoliageControllerComponent in the world.
type FoliageSystem struct{}

func NewFoliageSystem() *FoliageSystem {
	return &FoliageSystem{}
}

// Update simulates the foliage physics, applying wind effects to vertices.
func (s *FoliageSystem) Update(world *World, delta float64) {
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
	controller.windTime += delta

	foliageEntities := world.Query(CTFoliage, CTSprite, CTTransform)

	for _, entity := range foliageEntities {
		foliageComp, _ := world.GetComponent(entity, CTFoliage)
		foliage := foliageComp.(*FoliageComponent)

		spriteComp, _ := world.GetComponent(entity, CTSprite)
		sprite := spriteComp.(*SpriteComponent)

		if len(sprite.baseVertices) == 0 {
			continue
		}

		// Calculate height from base vertices for robustness
		minY, maxY := float32(math.Inf(1)), float32(math.Inf(-1))
		for _, v := range sprite.baseVertices {
			if v.DstY < minY {
				minY = v.DstY
			}
			if v.DstY > maxY {
				maxY = v.DstY
			}
		}
		height := float64(maxY - minY)
		if height == 0 {
			continue
		}

		// Default pivot to bottom center if not set
		pivot := foliage.Pivot
		if pivot.X == 0 && pivot.Y == 0 {
			pivot.X = 0.5
			pivot.Y = 1.0
		}

		// Calculate a directional sway factor (0 to 1) to simulate gusts
		sway := 0.5 + 0.5*math.Sin(controller.windTime*controller.windSpeed+foliage.SwaySeed)
		foliage.Angle = sway * controller.windForce

		// Reset vertices to base state before applying new transformations
		copy(sprite.Vertices, sprite.baseVertices)

		for i, baseVertex := range sprite.baseVertices {
			normalizedY := (baseVertex.DstY - minY) / float32(height)

			// The sway factor is based on the distance from the pivot.
			// It is 0 at the pivot and 1 at the point furthest from the pivot.
			dist := math.Abs(float64(normalizedY) - pivot.Y)
			maxDist := math.Max(pivot.Y, 1.0-pivot.Y)
			swayFactor := 0.0
			if maxDist > 0 {
				swayFactor = dist / maxDist
			}

			// Apply a power to make the bending more natural
			swayFactor = math.Pow(swayFactor, 1.5)

			angle := foliage.Angle * swayFactor

			// The displacement is proportional to the distance from the pivot
			displacementHeight := dist * height
			displacement := math.Sin(angle) * displacementHeight

			// Distribute the displacement along the wind direction vector
			displacementX := displacement * controller.windDirection.X
			displacementY := displacement * controller.windDirection.Y

			// Add the displacement to the base position to move it in the wind's direction
			sprite.Vertices[i].DstX = baseVertex.DstX + float32(displacementX)
			sprite.Vertices[i].DstY = baseVertex.DstY + float32(displacementY)
		}
	}
}
