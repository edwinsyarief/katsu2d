package katsu2d

import (
	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/lazyecs"
)

// MaxObjectsPerNode defines the maximum number of entities a node can hold before it subdivides.
const MaxObjectsPerNode = 10

// MaxDepth defines the maximum depth of the quadtree. This prevents infinite subdivision.
const MaxDepth = 8

// Quadtree is a data structure used to efficiently store and query entities in a 2D space.
// It partitions the space into smaller, rectangular nodes.
type Quadtree struct {
	// root is the top-level node of the quadtree.
	root *quadtreeNode
	// world is a reference to the main game world, used to access entity components.
	world *lazyecs.World
}

// NewQuadtree creates a new Quadtree instance, initializing it with a root node
// that covers the specified world bounds.
func NewQuadtree(world *lazyecs.World, bounds ebimath.Rectangle) *Quadtree {
	return &Quadtree{
		world: world,
		root:  newQuadtreeNode(world, bounds, 0),
	}
}

// Insert adds an entity to the quadtree. The entity is placed in the appropriate node
// based on its position.
func (self *Quadtree) Insert(obj lazyecs.Entity) {
	self.root.insert(obj)
}

// Query finds all entities within a given rectangular bounds.
// It returns a slice of entities that are located inside the query rectangle.
func (self *Quadtree) Query(bounds ebimath.Rectangle) []lazyecs.Entity {
	return self.root.query(bounds)
}

// QueryCircle finds all entities within a given circular area.
// It returns a slice of entities that are located inside the query circle.
func (self *Quadtree) QueryCircle(center ebimath.Vector, radius float64) []lazyecs.Entity {
	return self.root.queryCircle(center, radius)
}

// Clear resets the quadtree by creating a new empty root node with the same bounds.
func (self *Quadtree) Clear() {
	self.root = newQuadtreeNode(self.world, self.root.bounds, 0)
}

// quadtreeNode represents a single node in the quadtree.
// It can either contain entities or have four children nodes.
type quadtreeNode struct {
	// world is a reference to the game world.
	world *lazyecs.World
	// bounds is the rectangular area covered by this node.
	bounds ebimath.Rectangle
	// objects is a slice of entities contained directly within this node.
	objects []lazyecs.Entity
	// children are the four sub-nodes (top-left, top-right, bottom-left, bottom-right).
	// If a node has children, its objects slice is empty.
	children [4]*quadtreeNode
	// depth is the current depth of the node within the tree (root is depth 0).
	depth int
}

// newQuadtreeNode creates a new quadtreeNode with the specified bounds and depth.
func newQuadtreeNode(world *lazyecs.World, bounds ebimath.Rectangle, depth int) *quadtreeNode {
	return &quadtreeNode{
		world:   world,
		bounds:  bounds,
		objects: make([]lazyecs.Entity, 0),
		depth:   depth,
	}
}

// insert adds an entity to the current node. If the node exceeds the object limit and max depth,
// it subdivides and redistributes its entities into the new children nodes.
func (self *quadtreeNode) insert(e lazyecs.Entity) {
	// Get the TransformComponent to find the entity's position.
	t, ok := lazyecs.GetComponent[TransformComponent](self.world, e)
	if !ok {
		panic("Entity doesn't have TransformComponent")
	}

	// If the entity's position is not within the node's bounds, do nothing.
	if !self.bounds.Contains(t.Position()) {
		return
	}

	// If the node has children, pass the entity down to the correct child.
	if self.children[0] != nil {
		for _, child := range self.children {
			if child.bounds.Contains(t.Position()) {
				child.insert(e)
				return
			}
		}
		// If the entity is within the parent's bounds but not any of the children's,
		// it must be a boundary case. For this simple implementation, we just return.
		return
	}

	// Add the entity to the current node's object list.
	self.objects = append(self.objects, e)

	// Check if the node needs to subdivide.
	if len(self.objects) > MaxObjectsPerNode && self.depth < MaxDepth {
		// Subdivide the current node into four children.
		self.subdivide()

		// Move existing entities from the current node to the new children.
		for _, existingEntity := range self.objects {
			existingT, ok := lazyecs.GetComponent[TransformComponent](self.world, existingEntity)
			if !ok {
				panic("Entity doesn't have TransformComponent")
			}

			for _, child := range self.children {
				if child.bounds.Contains(existingT.Position()) {
					child.insert(existingEntity)
					break
				}
			}
		}
		// Clear the objects slice of the current node, as they have been moved to children.
		self.objects = make([]lazyecs.Entity, 0)
	}
}

// subdivide splits the current node into four smaller child nodes.
func (self *quadtreeNode) subdivide() {
	// Calculate the midpoint of the current node's bounds.
	midX := (self.bounds.Min.X + self.bounds.Max.X) / 2
	midY := (self.bounds.Min.Y + self.bounds.Max.Y) / 2

	// Create the four children nodes with their respective bounds.
	// Top-left child
	self.children[0] = newQuadtreeNode(self.world, ebimath.Rectangle{Min: self.bounds.Min, Max: ebimath.Vector{X: midX, Y: midY}}, self.depth+1)
	// Top-right child
	self.children[1] = newQuadtreeNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: self.bounds.Min.Y}, Max: ebimath.Vector{X: self.bounds.Max.X, Y: midY}}, self.depth+1)
	// Bottom-left child
	self.children[2] = newQuadtreeNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: self.bounds.Min.X, Y: midY}, Max: ebimath.Vector{X: midX, Y: self.bounds.Max.Y}}, self.depth+1)
	// Bottom-right child
	self.children[3] = newQuadtreeNode(self.world, ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: midY}, Max: self.bounds.Max}, self.depth+1)
}

// query recursively finds and returns entities within a given rectangular query area.
func (self *quadtreeNode) query(bounds ebimath.Rectangle) []lazyecs.Entity {
	var result []lazyecs.Entity

	// If the query rectangle does not intersect the current node's bounds,
	// no entities within this node or its children can be in the query area.
	if !self.bounds.Intersects(bounds) {
		return result
	}

	// If the node has no children, it's a leaf node. Check its objects.
	if self.children[0] == nil {
		for _, e := range self.objects {
			t, _ := lazyecs.GetComponent[TransformComponent](self.world, e)
			// Check if the entity's position is within the query bounds.
			if bounds.Contains(t.Position()) {
				result = append(result, e)
			}
		}
	} else {
		// If the node has children, recursively query each child that intersects the bounds.
		for _, child := range self.children {
			result = append(result, child.query(bounds)...)
		}
	}

	return result
}

// queryCircle recursively finds and returns entities within a given circular query area.
func (self *quadtreeNode) queryCircle(center ebimath.Vector, radius float64) []lazyecs.Entity {
	var result []lazyecs.Entity

	// If the query circle does not intersect the current node's bounds, return an empty slice.
	if !intersectsCircle(self.bounds, center, radius) {
		return result
	}

	// If the node has no children, it's a leaf node. Check its objects.
	if self.children[0] == nil {
		for _, e := range self.objects {
			t, _ := lazyecs.GetComponent[TransformComponent](self.world, e)
			// Check if the entity's position is within the query circle.
			if center.DistanceTo(t.Position()) <= radius {
				result = append(result, e)
			}
		}
	} else {
		// If the node has children, recursively query each child.
		for _, child := range self.children {
			result = append(result, child.queryCircle(center, radius)...)
		}
	}

	return result
}

// intersectsCircle checks if a rectangle and a circle overlap.
// This is used to determine if a node's bounds need to be checked during a circular query.
func intersectsCircle(rect ebimath.Rectangle, center ebimath.Vector, radius float64) bool {
	// Find the closest point on the rectangle to the circle's center.
	closestX := ebimath.Clamp(center.X, rect.Min.X, rect.Max.X)
	closestY := ebimath.Clamp(center.Y, rect.Min.Y, rect.Max.Y)
	// Calculate the distance squared from the circle's center to this closest point.
	distanceX := center.X - closestX
	distanceY := center.Y - closestY
	// The intersection occurs if the squared distance is less than the squared radius.
	return (distanceX*distanceX + distanceY*distanceY) < (radius * radius)
}
