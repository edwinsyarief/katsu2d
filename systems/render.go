package systems

import (
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"
	"katsu2d/graphics"

	"github.com/hajimehoshi/ebiten/v2"
)

// RenderSystem handles rendering of individual drawable components.
type RenderSystem struct {
	priority int
	renderer *graphics.Renderer
}

// NewRenderSystem creates a new RenderSystem instance.
func NewRenderSystem() *RenderSystem {
	return &RenderSystem{
		priority: 3,
		renderer: graphics.NewRenderer(),
	}
}

// Update is not implemented for the render system.
func (self *RenderSystem) Update(world *ecs.World, timeScale float64) error {
	return nil
}

// Draw draws all individual drawable entities.
func (self *RenderSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	// Get all entities with Drawable and Transform components
	entities := world.GetEntitiesWithComponents(constants.ComponentDrawable, constants.ComponentTransform)
	if len(entities) == 0 {
		return
	}

	// Begin a new batch for each texture, if needed. For this simple case, we use nil.
	self.renderer.Begin(nil)

	for _, entityID := range entities {
		drawableComp, hasDrawable := world.GetComponent(entityID, constants.ComponentDrawable)
		transformComp, hasTransform := world.GetComponent(entityID, constants.ComponentTransform)
		if !hasDrawable || !hasTransform {
			continue
		}

		drawable, ok1 := drawableComp.(*components.Drawable)
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

	// End the batch for the individual draw calls.
	self.renderer.End(screen)
}

// GetPriority returns the system's priority.
func (self *RenderSystem) GetPriority() int {
	return self.priority
}
