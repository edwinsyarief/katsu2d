package katsu2d

import ebimath "github.com/edwinsyarief/ebi-math"

const (
	MaxObjectsPerNode = 10
	MaxDepth          = 8
)

type QuadtreeECS struct {
	root  *quadtreeECSNode
	world *World
}

func NewQuadtreeECS(world *World, bounds ebimath.Rectangle) *QuadtreeECS {
	return &QuadtreeECS{
		world: world,
		root:  newQuadtreeECSNode(world, bounds, 0),
	}
}

func (self *QuadtreeECS) Insert(obj Entity) {
	self.root.insert(obj)
}

func (self *QuadtreeECS) Query(bounds ebimath.Rectangle) []Entity {
	return self.root.query(bounds)
}

func (self *QuadtreeECS) QueryCircle(center ebimath.Vector, radius float64) []Entity {
	return self.root.queryCircle(center, radius)
}

func (self *QuadtreeECS) Clear() {
	self.root = newQuadtreeECSNode(self.world, self.root.bounds, 0)
}

type quadtreeECSNode struct {
	world    *World
	bounds   ebimath.Rectangle
	objects  []Entity
	children [4]*quadtreeECSNode
	depth    int
}

func newQuadtreeECSNode(world *World, bounds ebimath.Rectangle, depth int) *quadtreeECSNode {
	return &quadtreeECSNode{
		world:   world,
		bounds:  bounds,
		objects: make([]Entity, 0),
		depth:   depth,
	}
}

func (self *quadtreeECSNode) insert(e Entity) {
	transform, ok := self.world.GetComponent(e, CTTransform)
	if !ok {
		panic("Entity doesn't have TransformComponent")
	}

	t := transform.(*TransformComponent)
	if !self.bounds.Contains(t.Position()) {
		return
	}

	if self.children[0] != nil {
		for _, child := range self.children {
			if child.bounds.Contains(t.Position()) {
				child.insert(e)
				return
			}
		}
		return
	}

	self.objects = append(self.objects, e)

	if len(self.objects) > MaxObjectsPerNode && self.depth < MaxDepth {
		self.subdivide()

		for _, existingEntity := range self.objects {
			existingTransform, ok := self.world.GetComponent(existingEntity, CTTransform)
			if !ok {
				panic("Entity doesn't have TransformComponent")
			}
			existingT := existingTransform.(*TransformComponent)
			for _, child := range self.children {
				if child.bounds.Contains(existingT.Position()) {
					child.insert(existingEntity)
					break
				}
			}
		}
		self.objects = make([]Entity, 0)
	}
}

func (self *quadtreeECSNode) subdivide() {
	midX := (self.bounds.Min.X + self.bounds.Max.X) / 2
	midY := (self.bounds.Min.Y + self.bounds.Max.Y) / 2

	self.children[0] = newQuadtreeECSNode(self.world, ebimath.Rectangle{Min: self.bounds.Min, Max: ebimath.Vector{X: midX, Y: midY}}, self.depth+1)
	self.children[1] = newQuadtreeECSNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: self.bounds.Min.Y}, Max: ebimath.Vector{X: self.bounds.Max.X, Y: midY}}, self.depth+1)
	self.children[2] = newQuadtreeECSNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: self.bounds.Min.X, Y: midY}, Max: ebimath.Vector{X: midX, Y: self.bounds.Max.Y}}, self.depth+1)
	self.children[3] = newQuadtreeECSNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: midY}, Max: self.bounds.Max}, self.depth+1)
}

func (self *quadtreeECSNode) query(bounds ebimath.Rectangle) []Entity {
	var result []Entity

	if !self.bounds.Intersects(bounds) {
		return result
	}

	if self.children[0] == nil {
		for _, e := range self.objects {
			transform, _ := self.world.GetComponent(e, CTTransform)
			t := transform.(*TransformComponent)
			if bounds.Contains(t.Position()) {
				result = append(result, e)
			}
		}
	} else {
		for _, child := range self.children {
			result = append(result, child.query(bounds)...)
		}
	}

	return result
}

func (self *quadtreeECSNode) queryCircle(center ebimath.Vector, radius float64) []Entity {
	var result []Entity

	if !intersectsCircle(self.bounds, center, radius) {
		return result
	}

	if self.children[0] == nil {
		for _, e := range self.objects {
			transform, _ := self.world.GetComponent(e, CTTransform)
			t := transform.(*TransformComponent)
			if center.DistanceTo(t.Position()) <= radius {
				result = append(result, e)
			}
		}
	} else {
		for _, child := range self.children {
			result = append(result, child.queryCircle(center, radius)...)
		}
	}

	return result
}

func intersectsCircle(rect ebimath.Rectangle, center ebimath.Vector, radius float64) bool {
	closestX := ebimath.Clamp(center.X, rect.Min.X, rect.Max.X)
	closestY := ebimath.Clamp(center.Y, rect.Min.Y, rect.Max.Y)
	distanceX := center.X - closestX
	distanceY := center.Y - closestY
	return (distanceX*distanceX + distanceY*distanceY) < (radius * radius)
}
