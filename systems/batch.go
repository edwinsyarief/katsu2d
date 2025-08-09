package systems

import (
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/ecs"
	"katsu2d/graphics"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// BatchSystem handles rendering batched sprites.
type BatchSystem struct {
	priority int
	renderer *graphics.Renderer
	textures map[int]*ebiten.Image // A map to hold all textures by their ID
}

// NewBatchSystem creates a new BatchSystem instance.
// It now takes a map of textures so it can access the images by ID.
func NewBatchSystem(textures map[int]*ebiten.Image) *BatchSystem {
	return &BatchSystem{
		priority: 2,
		renderer: graphics.NewRenderer(),
		textures: textures,
	}
}

// Update is not implemented for the batch system.
func (self *BatchSystem) Update(world *ecs.World, timeScale float64) error {
	return nil
}

// Draw draws all batched entities.
func (self *BatchSystem) Draw(world *ecs.World, screen *ebiten.Image) {
	currentTextureID := -1

	// Get all entities with DrawableBatch and Transform components
	for entityID := range world.GetEntitiesWithComponents(constants.ComponentDrawableBatch, constants.ComponentTransform) {
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

		t := transform.GetTransform()
		/* if transform.GetInitialParentTransform() != nil {
			t = transform.GetInitialParentTransform()
		} */

		realPos := ebimath.V2(0).Apply(t.Matrix())
		/* if !transform.Origin().IsZero() {
			realPos = realPos.Sub(transform.Origin())
		} */

		// Check if the texture ID has changed
		if drawable.TextureID != currentTextureID {
			// Flush the previous batch (if one exists)
			if currentTextureID != -1 {
				self.renderer.End(screen)
			}
			// Begin a new batch with the new texture
			if self.textures == nil {
				self.renderer.Begin(nil)
			} else {
				textureToUse := self.textures[drawable.TextureID]
				self.renderer.Begin(textureToUse)
			}
			currentTextureID = drawable.TextureID
		}

		// Check if the buffer is full before adding a new quad.
		if self.renderer.GetBatcher().IsFull(4) {
			currentTexture := self.textures[currentTextureID]
			// Flush the current batch and start a new one with the same texture.
			self.renderer.End(screen)
			self.renderer.Begin(currentTexture)
		}

		if drawable.TextureID <= 0 {
			// Draw a colored quad if no texture is specified
			self.renderer.GetBatcher().AddTransformedQuad(
				realPos.X, realPos.Y,
				drawable.Width,
				drawable.Height,
				transform.Scale().X, transform.Scale().Y,
				transform.Rotation(),
				0, 0, 1, 1,
				float64(drawable.Color.R)/255.0,
				float64(drawable.Color.G)/255.0,
				float64(drawable.Color.B)/255.0,
				float64(drawable.Color.A)/255.0,
			)
		} else {
			// Add the transformed quad to the batch
			self.renderer.GetBatcher().AddTransformedQuad(
				realPos.X, realPos.Y,
				drawable.Width,
				drawable.Height,
				transform.Scale().X, transform.Scale().Y,
				transform.Rotation(),
				drawable.SrcX, drawable.SrcY,
				drawable.SrcX+drawable.SrcW,
				drawable.SrcY+drawable.SrcH,
				float64(drawable.Color.R)/255.0,
				float64(drawable.Color.G)/255.0,
				float64(drawable.Color.B)/255.0,
				float64(drawable.Color.A)/255.0,
			)
		}
	}

	// End the batch for the individual draw calls.
	if currentTextureID != -1 {
		self.renderer.End(screen)
	}
}

// GetPriority returns the system's priority.
func (self *BatchSystem) GetPriority() int {
	return self.priority
}
