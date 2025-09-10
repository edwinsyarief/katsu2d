package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// FoliageSystem is the system responsible for updating and rendering foliage entities.
type FoliageSystem struct{}

// NewFoliageSystem creates and returns a new instance of the FoliageSystem.
func NewFoliageSystem() *FoliageSystem {
	return &FoliageSystem{}
}

// Update processes all foliage entities each frame, applying wind effects.
func (self *FoliageSystem) Update(world *World, dt float64) {
	// Retrieve the main foliage controller from the world.
	controllers := world.Query(CTFoliageController)
	if len(controllers) == 0 {
		return
	}

	controllerComp, ok := world.GetComponent(controllers[0], CTFoliageController)
	if !ok {
		return
	}
	controller := controllerComp.(*FoliageControllerComponent)
	// Increment the wind timer to progress the wind simulation.
	controller.windTime += dt

	// Iterate over all entities that have foliage, sprite, and transform components.
	for _, entity := range world.Query(CTFoliage, CTSprite, CTTransform) {
		foliageAny, _ := world.GetComponent(entity, CTFoliage)
		spriteAny, _ := world.GetComponent(entity, CTSprite)
		foliage := foliageAny.(*FoliageComponent)
		sprite := spriteAny.(*SpriteComponent)

		// Skip if the sprite has no base vertices to manipulate.
		if len(sprite.baseVertices) == 0 {
			continue
		}

		// Calculate the vertical bounds (minY and maxY) of the sprite's vertices.
		minY, maxY := getVertexBounds(sprite.baseVertices)
		height := float64(maxY - minY)
		if height == 0 {
			continue
		}

		// Determine the pivot point for rotation, using a default if none is specified.
		pivot := getDefaultPivot(foliage.Pivot)

		// Reset the sprite's current vertices to their original, base positions.
		copy(sprite.Vertices, sprite.baseVertices)

		// Apply the wind effect to each vertex based on the controller's state.
		applyWindEffect(sprite, controller, foliage, minY, height, pivot)
	}
}

// getVertexBounds calculates the minimum and maximum Y coordinates of a slice of vertices.
func getVertexBounds(vertices []ebiten.Vertex) (minY, maxY float32) {
	minY = float32(math.Inf(1))
	maxY = float32(math.Inf(-1))
	for _, v := range vertices {
		minY = min(minY, v.DstY)
		maxY = max(maxY, v.DstY)
	}
	return
}

// getDefaultPivot returns a default pivot point (bottom center) if the provided pivot is zero.
func getDefaultPivot(pivot ebimath.Vector) ebimath.Vector {
	if pivot.X == 0 && pivot.Y == 0 {
		return ebimath.V(0.5, 1.0) // Default pivot at the bottom center
	}
	return pivot
}

// applyWindEffect calculates and applies wind-induced displacement to the foliage's vertices.
// The effect combines a smooth sine wave with Perlin noise for a more natural look.
func applyWindEffect(sprite *SpriteComponent, controller *FoliageControllerComponent,
	foliage *FoliageComponent, minY float32, height float64, pivot ebimath.Vector) {

	// Add a small amount of variation to the sway frequency for each foliage instance.
	swayFreq := 1.0 + controller.noise.Noise3D(foliage.SwaySeed, 0, 0)*0.2

	for i, baseVertex := range sprite.baseVertices {
		// Calculate a normalized Y-coordinate (0.0 at the top, 1.0 at the bottom).
		normalizedY := (baseVertex.DstY - minY) / float32(height)
		// The sway factor is based on how far the vertex is from the pivot point.
		// A higher power (1.5) makes the tip sway more dramatically than the base.
		swayFactor := math.Pow(math.Abs(float64(normalizedY)-pivot.Y), 1.5)

		// Sample Perlin noise to add randomness to the sway.
		swayNoise := controller.noise.Noise3D(controller.windTime*controller.windSpeed*0.1*swayFreq+foliage.SwaySeed, 0, 0)

		// Primary ripple: a smooth wave traveling with the wind direction.
		// The dot product projects the vertex position onto the wind direction vector.
		dotProduct := controller.windDirection.X*float64(baseVertex.DstX) + controller.windDirection.Y*float64(baseVertex.DstY)
		ripplePhase := controller.windTime*controller.windSpeed*2.0 - dotProduct*0.05
		primaryRipple := math.Sin(ripplePhase)

		// Secondary ripple: slower, more subtle Perlin noise to add randomness to the overall ripple.
		secondaryRipple := controller.noise.Noise3D(controller.windTime*controller.windSpeed*0.5, float64(baseVertex.DstX)*0.02, float64(baseVertex.DstY)*0.02)

		// Combine the two ripple noise sources with different weights.
		rippleNoise := primaryRipple*0.7 + secondaryRipple*0.3

		// Calculate the final sway value, combining a sine wave and Perlin noise.
		swayValue := math.Sin(controller.windTime*controller.windSpeed*0.1*swayFreq + foliage.SwaySeed + swayNoise*10)

		// Total displacement is a combination of the overall wind force and the ripple effects.
		displacement := (swayValue*0.25+0.75)*controller.windForce + rippleNoise*controller.rippleStrength

		// Calculate the final displacement in X and Y directions, taking wind direction and sway factor into account.
		dispX := calculateDisplacement(displacement, swayFactor, controller.windDirection.X)
		dispY := calculateDisplacement(displacement, swayFactor, controller.windDirection.Y)

		// Apply the calculated displacement to the sprite's vertices.
		sprite.Vertices[i].DstX = baseVertex.DstX + float32(dispX)
		sprite.Vertices[i].DstY = baseVertex.DstY + float32(dispY)
	}
}

// calculateDisplacement ensures the displacement only occurs in the direction of the wind.
// For example, a positive wind direction only results in a positive displacement.
func calculateDisplacement(displacement, swayFactor, direction float64) float64 {
	disp := displacement * swayFactor * direction
	if direction > 0 {
		return math.Max(0, disp)
	} else if direction < 0 {
		return math.Min(0, disp)
	}
	return disp
}
