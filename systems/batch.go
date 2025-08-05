package systems

import (
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"
	"katsu2d/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// BatchSystem handles rendering batched sprites.
type BatchSystem struct {
	priority int
	renderer *graphics.Renderer
}

// NewBatchSystem creates a new BatchSystem instance.
func NewBatchSystem() *BatchSystem {
	return &BatchSystem{
		priority: 2,
		renderer: graphics.NewRenderer(),
	}
}

// Update is not implemented for the batch system.
func (self *BatchSystem) Update(world *ecs.World, timeScale float64) error {
	// Batch system updates during draw
	return nil
}

// Draw draws all batched entities.
func (self *BatchSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Get all entities with DrawableBatch and Transform components
	entities := world.GetEntitiesWithComponents(constants.ComponentDrawableBatch, constants.ComponentTransform)
	if len(entities) == 0 {
		return
	}

	// Begin batching
	self.renderer.Begin(nil) // Using white pixel texture for color

	for _, entityID := range entities {
		drawableComp, hasDrawable := world.GetComponent(entityID, constants.ComponentDrawableBatch)
		transformComp, hasTransform := world.GetComponent(entityID, constants.ComponentTransform)
		if !hasDrawable || !hasTransform {
			continue
		}

		drawable, ok1 := drawableComp.(*components.DrawableBatch)
		transform, ok2 := transformComp.(*components.Transform)
		if !ok1 || !ok2 {
			continue
		}

		// Draw the quad with rotation and scaling
		pos := transform.Position()
		scale := transform.Scale()
		rotation := transform.Rotation()
		self.renderer.DrawTransformedQuad(
			pos.X,
			pos.Y,
			drawable.Width,
			drawable.Height,
			scale.X,
			scale.Y,
			rotation,
			drawable.Color,
		)
	}

	// End batching and flush
	self.renderer.End(screen)
}

// GetPriority returns the system's priority.
func (self *BatchSystem) GetPriority() int {
	return self.priority
}
