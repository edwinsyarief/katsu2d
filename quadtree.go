package katsu2d

import (
	"sync"

	"github.com/edwinsyarief/teishoku"
)

// MaxObjectsPerNode defines the maximum number of entities a node can hold before it subdivides.
const MaxObjectsPerNode = 16 // Increased from 10 for fewer subdivisions/depth.

// MaxDepth defines the maximum depth of the quadtree. This prevents infinite subdivision.
const MaxDepth = 10 // Increased from 8 for slightly deeper trees if needed.

var nodePool = sync.Pool{
	New: func() interface{} {
		return &quadtreeNode{
			objects: make([]teishoku.Entity, 0, MaxObjectsPerNode+1),
		}
	},
}

// Quadtree is a data structure used to efficiently store and query entities in a 2D space.
// It partitions the space into smaller, rectangular nodes.
type Quadtree struct {
	// root is the top-level node of the quadtree.
	root *quadtreeNode
	// world is a reference to the main game world, used to access entity components.
	world *teishoku.World
	// builder is shared among all nodes to access TransformComponent.
	builder *teishoku.Builder[TransformComponent]
}

// NewQuadtree creates a new Quadtree instance, initializing it with a root node
// that covers the specified world bounds.
func NewQuadtree(world *teishoku.World, bounds Rectangle) *Quadtree {
	res := &Quadtree{
		world:   world,
		root:    newQuadtreeNode(bounds, 0),
		builder: teishoku.NewBuilder[TransformComponent](world),
	}
	res.root.builder = res.builder
	return res
}

// Insert adds an entity to the quadtree. The entity is placed in the appropriate node
// based on its position.
func (self *Quadtree) Insert(obj teishoku.Entity) {
	self.root.insert(obj)
}

// Query finds all entities within a given rectangular bounds.
// It returns a slice of entities that are located inside the query rectangle.
func (self *Quadtree) Query(bounds Rectangle) []teishoku.Entity {
	var result []teishoku.Entity
	self.root.query(bounds, &result)
	return result
}

// QueryCircle finds all entities within a given circular area.
// It returns a slice of entities that are located inside the query circle.
func (self *Quadtree) QueryCircle(center Vector, radius float64) []teishoku.Entity {
	var result []teishoku.Entity
	self.root.queryCircle(center, radius, &result)
	return result
}

// Clear resets the quadtree by releasing the old root and creating a new empty root node with the same bounds.
func (self *Quadtree) Clear() {
	if self.root != nil {
		self.root.release()
	}
	self.root = newQuadtreeNode(self.root.bounds, 0)
	self.root.builder = self.builder
}

// quadtreeNode represents a single node in the quadtree.
// It can either contain entities or have four children nodes.
type quadtreeNode struct {
	// children are the four sub-nodes (top-left, top-right, bottom-left, bottom-right).
	// If a node has children, its objects slice is empty.
	children [4]*quadtreeNode
	// builder is used to access TransformComponent of entities.
	builder *teishoku.Builder[TransformComponent]
	// objects is a slice of entities contained directly within this node.
	objects []teishoku.Entity
	// bounds is the rectangular area covered by this node.
	bounds Rectangle
	// midX, midY are precomputed midpoints for fast child index computation.
	midX, midY float64
	// depth is the current depth of the node within the tree (root is depth 0).
	depth int
}

// newQuadtreeNode creates a new quadtreeNode with the specified bounds and depth.
func newQuadtreeNode(bounds Rectangle, depth int) *quadtreeNode {
	n := nodePool.Get().(*quadtreeNode)
	n.bounds = bounds
	n.depth = depth
	n.objects = n.objects[:0]
	n.midX = 0
	n.midY = 0
	for i := range n.children {
		n.children[i] = nil
	}
	return n
}

// release returns the node and its subtree to the pool.
func (self *quadtreeNode) release() {
	for _, child := range self.children {
		if child != nil {
			child.release()
		}
	}
	self.objects = self.objects[:0]
	for i := range self.children {
		self.children[i] = nil
	}
	nodePool.Put(self)
}

// insert adds an entity to the current node. If the node exceeds the object limit and max depth,
// it subdivides and redistributes its entities into the new children nodes.
func (self *quadtreeNode) insert(e teishoku.Entity) {
	// Get the TransformComponent to find the entity's position.
	t := self.builder.Get(e)
	if t == nil {
		panic("Entity doesn't have TransformComponent")
	}
	pos := Vector(t.Position)
	// If the entity's position is not within the node's bounds, do nothing.
	if !self.bounds.Contains(pos) {
		return
	}
	// If the node has children, pass the entity down to the correct child using computed index.
	if self.children[0] != nil {
		childIdx := 0
		if pos.X >= self.midX {
			childIdx |= 1
		}
		if pos.Y >= self.midY {
			childIdx |= 2
		}
		self.children[childIdx].insert(e)
		return
	}
	// Add the entity to the current node's object list.
	self.objects = append(self.objects, e)
	// Check if the node needs to subdivide.
	if len(self.objects) > MaxObjectsPerNode && self.depth < MaxDepth {
		// Subdivide the current node into four children.
		self.subdivide()
		// Move existing entities from the current node to the new children using computed index.
		for _, existingEntity := range self.objects {
			existingT := self.builder.Get(existingEntity)
			if existingT == nil {
				panic("Entity doesn't have TransformComponent")
			}
			existingPos := existingT.Position
			childIdx := 0
			if existingPos.X >= self.midX {
				childIdx |= 1
			}
			if existingPos.Y >= self.midY {
				childIdx |= 2
			}
			self.children[childIdx].insert(existingEntity)
		}
		// Clear the objects slice of the current node, as they have been moved to children.
		self.objects = self.objects[:0]
	}
}

// subdivide splits the current node into four smaller child nodes.
func (self *quadtreeNode) subdivide() {
	// Calculate the midpoint of the current node's bounds.
	self.midX = (self.bounds.Min.X + self.bounds.Max.X) / 2
	self.midY = (self.bounds.Min.Y + self.bounds.Max.Y) / 2
	halfW := (self.bounds.Max.X - self.bounds.Min.X) / 2
	halfH := (self.bounds.Max.Y - self.bounds.Min.Y) / 2
	// Create the four children nodes with their respective bounds.
	// Top-left child (index 0: X < midX, Y < midY)
	self.children[0] = newQuadtreeNode(Rectangle{Min: self.bounds.Min, Max: Vector{X: self.midX, Y: self.midY}}, self.depth+1)
	self.children[0].builder = self.builder
	// Top-right child (index 1: X >= midX, Y < midY)
	self.children[1] = newQuadtreeNode(Rectangle{Min: Vector{X: self.midX, Y: self.bounds.Min.Y}, Max: Vector{X: self.midX + halfW, Y: self.midY}}, self.depth+1)
	self.children[1].builder = self.builder
	// Bottom-left child (index 2: X < midX, Y >= midY)
	self.children[2] = newQuadtreeNode(Rectangle{Min: Vector{X: self.bounds.Min.X, Y: self.midY}, Max: Vector{X: self.midX, Y: self.midY + halfH}}, self.depth+1)
	self.children[2].builder = self.builder
	// Bottom-right child (index 3: X >= midX, Y >= midY)
	self.children[3] = newQuadtreeNode(Rectangle{Min: Vector{X: self.midX, Y: self.midY}, Max: self.bounds.Max}, self.depth+1)
	self.children[3].builder = self.builder
}

// query recursively finds and appends entities within a given rectangular query area to the result slice.
func (self *quadtreeNode) query(bounds Rectangle, result *[]teishoku.Entity) {
	// If the query rectangle does not intersect the current node's bounds,
	// no entities within this node or its children can be in the query area.
	if !self.bounds.Intersects(bounds) {
		return
	}
	// If the node has no children, it's a leaf node. Check its objects.
	if self.children[0] == nil {
		for _, e := range self.objects {
			t := self.builder.Get(e)
			// Check if the entity's position is within the query bounds.
			if bounds.Contains(Vector(t.Position)) {
				*result = append(*result, e)
			}
		}
	} else {
		// If the node has children, recursively query each child (no pruning for uniform data benchmarks).
		for _, child := range self.children {
			child.query(bounds, result)
		}
	}
}

// queryCircle recursively finds and appends entities within a given circular query area to the result slice.
func (self *quadtreeNode) queryCircle(center Vector, radius float64, result *[]teishoku.Entity) {
	// If the query circle does not intersect the current node's bounds, return.
	if !intersectsCircle(self.bounds, center, radius) {
		return
	}
	// If the node has no children, it's a leaf node. Check its objects.
	if self.children[0] == nil {
		for _, e := range self.objects {
			t := self.builder.Get(e)
			// Check if the entity's position is within the query circle.
			if center.DistanceTo(Vector(t.Position)) <= radius {
				*result = append(*result, e)
			}
		}
	} else {
		// If the node has children, recursively query each child (no pruning for uniform data benchmarks).
		for _, child := range self.children {
			child.queryCircle(center, radius, result)
		}
	}
}

// intersectsCircle checks if a rectangle and a circle overlap.
// This is used to determine if a node's bounds need to be checked during a circular query.
func intersectsCircle(rect Rectangle, center Vector, radius float64) bool {
	// Find the closest point on the rectangle to the circle's center.
	closestX := Clamp(center.X, rect.Min.X, rect.Max.X)
	closestY := Clamp(center.Y, rect.Min.Y, rect.Max.Y)
	// Calculate the distance squared from the circle's center to this closest point.
	distanceX := center.X - closestX
	distanceY := center.Y - closestY
	// The intersection occurs if the squared distance is less than the squared radius.
	return (distanceX*distanceX + distanceY*distanceY) < (radius * radius)
}
