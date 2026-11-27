package katsu2d

import (
	"sync"

	"github.com/edwinsyarief/teishoku"
)

// MaxObjectsPerNode defines the maximum number of entities a node holds before subdividing.
const MaxObjectsPerNode = 16

// MaxDepth defines the maximum subdivision depth of the quadtree.
const MaxDepth = 10

var nodePool = sync.Pool{
	New: func() any {
		return &quadtreeNode{
			objects: make([]teishoku.Entity, 0, MaxObjectsPerNode+1),
		}
	},
}

// Quadtree partitions 2D space into recursive rectangular nodes for fast spatial queries.
type Quadtree struct {
	root    *quadtreeNode
	world   *teishoku.World
	builder *teishoku.Builder[TransformComponent]
	bounds  Rectangle
}

// NewQuadtree creates a new Quadtree covering the specified world bounds.
func NewQuadtree(world *teishoku.World, bounds Rectangle) *Quadtree {
	b := teishoku.NewBuilder[TransformComponent](world)
	root := newQuadtreeNode(bounds, 0, b)
	return &Quadtree{
		world:   world,
		root:    root,
		builder: b,
		bounds:  bounds,
	}
}

// Bounds returns the root rectangular bounds of the quadtree.
func (q *Quadtree) Bounds() Rectangle {
	return q.bounds
}

// Insert adds an entity with a TransformComponent to the quadtree.
func (q *Quadtree) Insert(e teishoku.Entity) {
	t := q.builder.Get(e)
	if t == nil {
		return
	}
	q.root.insert(e, Vector(t.Position))
}

// Query returns all entities within the given rectangular bounds.
func (q *Quadtree) Query(bounds Rectangle) []teishoku.Entity {
	return q.QueryInto(bounds, nil)
}

// QueryInto appends all entities within bounds into dst (reusing dst's capacity) and returns it.
func (q *Quadtree) QueryInto(bounds Rectangle, dst []teishoku.Entity) []teishoku.Entity {
	dst = dst[:0]
	q.root.query(bounds, &dst)
	return dst
}

// QueryCircle returns all entities within the given circular area.
func (q *Quadtree) QueryCircle(center Vector, radius float64) []teishoku.Entity {
	return q.QueryCircleInto(center, radius, nil)
}

// QueryCircleInto appends all entities within the circle into dst (reusing dst's capacity) and returns it.
func (q *Quadtree) QueryCircleInto(center Vector, radius float64, dst []teishoku.Entity) []teishoku.Entity {
	dst = dst[:0]
	q.root.queryCircle(center, radius, radius*radius, &dst)
	return dst
}

// Clear releases all nodes back to the pool and resets the quadtree to an empty root.
func (q *Quadtree) Clear() {
	if q.root != nil {
		q.root.release()
	}
	q.root = newQuadtreeNode(q.bounds, 0, q.builder)
}

type quadtreeNode struct {
	children [4]*quadtreeNode
	builder  *teishoku.Builder[TransformComponent]
	objects  []teishoku.Entity
	bounds   Rectangle
	midX     float64
	midY     float64
	depth    int
}

func newQuadtreeNode(bounds Rectangle, depth int, builder *teishoku.Builder[TransformComponent]) *quadtreeNode {
	n := nodePool.Get().(*quadtreeNode)
	n.bounds = bounds
	n.depth = depth
	n.builder = builder
	n.objects = n.objects[:0]
	n.midX = 0
	n.midY = 0
	n.children = [4]*quadtreeNode{}
	return n
}

func (n *quadtreeNode) release() {
	for i, child := range n.children {
		if child != nil {
			child.release()
			n.children[i] = nil
		}
	}
	n.objects = n.objects[:0]
	n.builder = nil
	nodePool.Put(n)
}

func (n *quadtreeNode) childIndex(pos Vector) int {
	idx := 0
	if pos.X >= n.midX {
		idx |= 1
	}
	if pos.Y >= n.midY {
		idx |= 2
	}
	return idx
}

func (n *quadtreeNode) insert(e teishoku.Entity, pos Vector) {
	if !n.bounds.Contains(pos) {
		return
	}
	if n.children[0] != nil {
		n.children[n.childIndex(pos)].insert(e, pos)
		return
	}

	n.objects = append(n.objects, e)
	if len(n.objects) > MaxObjectsPerNode && n.depth < MaxDepth {
		n.subdivide()
		for _, existing := range n.objects {
			if existingT := n.builder.Get(existing); existingT != nil {
				existingPos := Vector(existingT.Position)
				n.children[n.childIndex(existingPos)].insert(existing, existingPos)
			}
		}
		n.objects = n.objects[:0]
	}
}

func (n *quadtreeNode) subdivide() {
	n.midX = (n.bounds.Min.X + n.bounds.Max.X) * 0.5
	n.midY = (n.bounds.Min.Y + n.bounds.Max.Y) * 0.5
	nextDepth := n.depth + 1

	n.children[0] = newQuadtreeNode(Rectangle{Min: n.bounds.Min, Max: Vector{X: n.midX, Y: n.midY}}, nextDepth, n.builder)
	n.children[1] = newQuadtreeNode(Rectangle{Min: Vector{X: n.midX, Y: n.bounds.Min.Y}, Max: Vector{X: n.bounds.Max.X, Y: n.midY}}, nextDepth, n.builder)
	n.children[2] = newQuadtreeNode(Rectangle{Min: Vector{X: n.bounds.Min.X, Y: n.midY}, Max: Vector{X: n.midX, Y: n.bounds.Max.Y}}, nextDepth, n.builder)
	n.children[3] = newQuadtreeNode(Rectangle{Min: Vector{X: n.midX, Y: n.midY}, Max: n.bounds.Max}, nextDepth, n.builder)
}

func (n *quadtreeNode) query(bounds Rectangle, result *[]teishoku.Entity) {
	if !n.bounds.Intersects(bounds) {
		return
	}
	if n.children[0] == nil {
		for _, e := range n.objects {
			if t := n.builder.Get(e); t != nil && bounds.Contains(Vector(t.Position)) {
				*result = append(*result, e)
			}
		}
		return
	}
	for _, child := range n.children {
		child.query(bounds, result)
	}
}

func (n *quadtreeNode) queryCircle(center Vector, radius, radiusSq float64, result *[]teishoku.Entity) {
	if !n.bounds.IntersectsCircle(center, radius) {
		return
	}
	if n.children[0] == nil {
		for _, e := range n.objects {
			if t := n.builder.Get(e); t != nil && center.DistanceSquaredTo(Vector(t.Position)) <= radiusSq {
				*result = append(*result, e)
			}
		}
		return
	}
	for _, child := range n.children {
		child.queryCircle(center, radius, radiusSq, result)
	}
}
