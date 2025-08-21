package katsu2d

import (
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

type FoliageSystem struct{}

func NewFoliageSystem() *FoliageSystem {
	return &FoliageSystem{}
}

func (s *FoliageSystem) Update(world *World, delta float64) {
	// Get controller
	controllers := world.Query(CTFoliageController)
	if len(controllers) == 0 {
		return
	}

	controllerComp, ok := world.GetComponent(controllers[0], CTFoliageController)
	if !ok {
		return
	}
	controller := controllerComp.(*FoliageControllerComponent)
	controller.windTime += delta

	// Process foliage entities
	for _, entity := range world.Query(CTFoliage, CTSprite, CTTransform) {
		foliageAny, _ := world.GetComponent(entity, CTFoliage)
		spriteAny, _ := world.GetComponent(entity, CTSprite)
		foliage := foliageAny.(*FoliageComponent)
		sprite := spriteAny.(*SpriteComponent)

		if len(sprite.baseVertices) == 0 {
			continue
		}

		// Calculate sprite dimensions
		minY, maxY := getVertexBounds(sprite.baseVertices)
		height := float64(maxY - minY)
		if height == 0 {
			continue
		}

		// Set default pivot if not specified
		pivot := getDefaultPivot(foliage.Pivot)

		// Reset vertices to base state
		copy(sprite.Vertices, sprite.baseVertices)

		// Apply wind effect to each vertex
		applyWindEffect(sprite, controller, foliage, minY, height, pivot)
	}
}

func getVertexBounds(vertices []ebiten.Vertex) (minY, maxY float32) {
	minY = float32(math.Inf(1))
	maxY = float32(math.Inf(-1))
	for _, v := range vertices {
		minY = min(minY, v.DstY)
		maxY = max(maxY, v.DstY)
	}
	return
}

func getDefaultPivot(pivot ebimath.Vector) ebimath.Vector {
	if pivot.X == 0 && pivot.Y == 0 {
		return ebimath.V(0.5, 1.0) // Default pivot at the bottom center
	}
	return pivot
}

func applyWindEffect(sprite *SpriteComponent, controller *FoliageControllerComponent,
	foliage *FoliageComponent, minY float32, height float64, pivot ebimath.Vector) {

	for i, baseVertex := range sprite.baseVertices {
		normalizedY := (baseVertex.DstY - minY) / float32(height)
		swayFactor := math.Pow(math.Abs(float64(normalizedY)-pivot.Y), 1.5)

		// Calculate wind displacement
		swayNoise := controller.noise.Noise3D(controller.windTime*controller.windSpeed*0.1+foliage.SwaySeed, 0, 0)
		rippleNoise := controller.noise.Noise3D(controller.windTime*controller.windSpeed+float64(baseVertex.DstX)*0.05, 0, 0)
		swayValue := math.Sin(controller.windTime*controller.windSpeed*0.1 + swayNoise*10)

		displacement := (swayValue * controller.windForce) + (rippleNoise * controller.rippleStrength)

		// Calculate final displacement
		dispX := calculateDisplacement(displacement, swayFactor, controller.windDirection.X)
		dispY := calculateDisplacement(displacement, swayFactor, controller.windDirection.Y)

		// Apply displacement
		sprite.Vertices[i].DstX = baseVertex.DstX + float32(dispX)
		sprite.Vertices[i].DstY = baseVertex.DstY + float32(dispY)
	}
}

func calculateDisplacement(displacement, swayFactor, direction float64) float64 {
	disp := displacement * swayFactor * direction
	if direction > 0 {
		return math.Abs(disp)
	}
	if direction < 0 && disp > 0 {
		return -disp
	}
	return disp
}
