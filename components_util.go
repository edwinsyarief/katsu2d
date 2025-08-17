package katsu2d

import ebimath "github.com/edwinsyarief/ebi-math"

type TagComponent struct {
	Tag string
}

func NewTagComponent(tag string) *TagComponent {
	return &TagComponent{Tag: tag}
}

type TransformComponent struct {
	*ebimath.Transform
	Z float64
}

func (self *TransformComponent) Position() ebimath.Vector {
	return self.Transform.Position()
}

func NewTransformComponent() *TransformComponent {
	return &TransformComponent{
		Transform: ebimath.T(),
	}
}

func (self *TransformComponent) SetZ(world *World, z float64) {
	if self.Z != z {
		self.Z = z
		world.MarkZDirty()
	}
}
