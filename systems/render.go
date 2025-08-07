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
	textures map[int]*ebiten.Image // A map to hold all textures by their ID
}

// NewRenderSystem creates a new RenderSystem instance.
// It now takes a map of textures so it can access the images by ID.
func NewRenderSystem(textures map[int]*ebiten.Image) *RenderSystem {
	return &RenderSystem{
		priority: 3,
		renderer: graphics.NewRenderer(),
		textures: textures,
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

		// Get the correct texture from the map using the component's TextureID
		if self.textures == nil {
			self.renderer.Begin(nil)
		} else {
			textureToUse := self.textures[drawable.TextureID]
			// Begin and end the batch for a single draw call. This is inefficient but necessary
			// for a non-batched render system that uses the batcher.
			self.renderer.Begin(textureToUse)
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

		// End the batch to render the single quad.
		self.renderer.End(screen)
	}
}

// GetPriority returns the system's priority.
func (self *RenderSystem) GetPriority() int {
	return self.priority
}
