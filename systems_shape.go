package katsu2d

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// ShapeRenderSystem renders shape components.
type ShapeRenderSystem struct {
	img *ebiten.Image
}

// NewShapeRenderSystem creates a new ShapeRenderSystem.
func NewShapeRenderSystem() *ShapeRenderSystem {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return &ShapeRenderSystem{
		img: img,
	}
}

// Update rebuilds the mesh for any dirty shapes.
func (self *ShapeRenderSystem) Update(world *World, dt float64) {
	self.updateShapes(world, CTRectangle)
	self.updateShapes(world, CTCircle)
	self.updateShapes(world, CTTriangle)
	self.updateShapes(world, CTPentagon)
	self.updateShapes(world, CTHexagon)
}

func (self *ShapeRenderSystem) updateShapes(world *World, componentID ComponentID) {
	entities := world.Query(componentID)
	for _, e := range entities {
		shapeAny, _ := world.GetComponent(e, componentID)
		shape := shapeAny.(Shape)
		shape.Rebuild()
	}
}

// Draw renders all shape components in the world.
func (self *ShapeRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	// Handle RectangleComponent separately
	rectEntities := world.Query(CTTransform, CTRectangle)
	for _, e := range rectEntities {
		rectAny, _ := world.GetComponent(e, CTRectangle)
		rect := rectAny.(*RectangleComponent)

		tAny, _ := world.GetComponent(e, CTTransform)
		t := tAny.(*TransformComponent)

		if len(rect.Vertices) == 0 {
			continue
		}

		worldVertices := make([]ebiten.Vertex, len(rect.Vertices))
		transformMatrix := t.Matrix()
		for i, v := range rect.Vertices {
			vx, vy := transformMatrix.Apply(float64(v.DstX), float64(v.DstY))
			v.DstX = float32(vx)
			v.DstY = float32(vy)
			v.SrcX = 0
			v.SrcY = 0
			worldVertices[i] = v
		}
		renderer.AddCustomMeshes(worldVertices, rect.Indices, self.img)
	}

	// Handle other shapes
	self.drawShapes(world, renderer, CTTransform, CTCircle)
	self.drawShapes(world, renderer, CTTransform, CTTriangle)
	self.drawShapes(world, renderer, CTTransform, CTPentagon)
	self.drawShapes(world, renderer, CTTransform, CTHexagon)
}

func (self *ShapeRenderSystem) drawShapes(world *World, renderer *BatchRenderer, transformID ComponentID, shapeID ComponentID) {
	entities := world.Query(transformID, shapeID)
	for _, e := range entities {
		shapeAny, _ := world.GetComponent(e, shapeID)
		shape := shapeAny.(Shape)

		tAny, _ := world.GetComponent(e, transformID)
		t := tAny.(*TransformComponent)

		vertices := shape.GetVertices()
		indices := shape.GetIndices()

		if len(vertices) == 0 {
			continue
		}

		worldVertices := make([]ebiten.Vertex, len(vertices))
		transformMatrix := t.Matrix()
		for i, v := range vertices {
			vx, vy := transformMatrix.Apply(float64(v.DstX), float64(v.DstY))
			v.DstX = float32(vx)
			v.DstY = float32(vy)
			v.SrcX = 0
			v.SrcY = 0
			worldVertices[i] = v
		}
		renderer.AddCustomMeshes(worldVertices, indices, self.img)
	}
}
