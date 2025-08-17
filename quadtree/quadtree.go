package quadtree

import (
	ebimath "github.com/edwinsyarief/ebi-math"
)

const (
	MaxObjectsPerNode = 10
	MaxDepth          = 8
)

type Object interface {
	Position() ebimath.Vector
}

type Quadtree struct {
	root *quadtreeNode
}

func New(bounds ebimath.Rectangle) *Quadtree {
	return &Quadtree{
		root: newQuadtreeNode(bounds, 0),
	}
}

func (qt *Quadtree) Insert(obj Object) {
	qt.root.insert(obj)
}

func (qt *Quadtree) Query(bounds ebimath.Rectangle) []Object {
	return qt.root.query(bounds)
}

func (qt *Quadtree) QueryCircle(center ebimath.Vector, radius float64) []Object {
	return qt.root.queryCircle(center, radius)
}

func (qt *Quadtree) Clear() {
	qt.root = newQuadtreeNode(qt.root.bounds, 0)
}

type quadtreeNode struct {
	bounds   ebimath.Rectangle
	objects  []Object
	children [4]*quadtreeNode
	depth    int
}

func newQuadtreeNode(bounds ebimath.Rectangle, depth int) *quadtreeNode {
	return &quadtreeNode{
		bounds:  bounds,
		objects: make([]Object, 0),
		depth:   depth,
	}
}

func (qn *quadtreeNode) insert(obj Object) {
	if !qn.bounds.Contains(obj.Position()) {
		return
	}

	if qn.children[0] != nil {
		for _, child := range qn.children {
			if child.bounds.Contains(obj.Position()) {
				child.insert(obj)
				return
			}
		}
		return
	}

	qn.objects = append(qn.objects, obj)

	if len(qn.objects) > MaxObjectsPerNode && qn.depth < MaxDepth {
		qn.subdivide()

		for _, existingObj := range qn.objects {
			for _, child := range qn.children {
				if child.bounds.Contains(existingObj.Position()) {
					child.insert(existingObj)
					break
				}
			}
		}
		qn.objects = make([]Object, 0)
	}
}

func (qn *quadtreeNode) subdivide() {
	midX := (qn.bounds.Min.X + qn.bounds.Max.X) / 2
	midY := (qn.bounds.Min.Y + qn.bounds.Max.Y) / 2

	qn.children[0] = newQuadtreeNode(ebimath.Rectangle{Min: qn.bounds.Min, Max: ebimath.Vector{X: midX, Y: midY}}, qn.depth+1)
	qn.children[1] = newQuadtreeNode(ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: qn.bounds.Min.Y}, Max: ebimath.Vector{X: qn.bounds.Max.X, Y: midY}}, qn.depth+1)
	qn.children[2] = newQuadtreeNode(ebimath.Rectangle{Min: ebimath.Vector{X: qn.bounds.Min.X, Y: midY}, Max: ebimath.Vector{X: midX, Y: qn.bounds.Max.Y}}, qn.depth+1)
	qn.children[3] = newQuadtreeNode(ebimath.Rectangle{Min: ebimath.Vector{X: midX, Y: midY}, Max: qn.bounds.Max}, qn.depth+1)
}

func (qn *quadtreeNode) query(bounds ebimath.Rectangle) []Object {
	var result []Object

	if !qn.bounds.Intersects(bounds) {
		return result
	}

	if qn.children[0] == nil {
		for _, obj := range qn.objects {
			if bounds.Contains(obj.Position()) {
				result = append(result, obj)
			}
		}
	} else {
		for _, child := range qn.children {
			result = append(result, child.query(bounds)...)
		}
	}

	return result
}

func (qn *quadtreeNode) queryCircle(center ebimath.Vector, radius float64) []Object {
	var result []Object

	if !intersectsCircle(qn.bounds, center, radius) {
		return result
	}

	if qn.children[0] == nil {
		for _, obj := range qn.objects {
			if center.DistanceTo(obj.Position()) <= radius {
				result = append(result, obj)
			}
		}
	} else {
		for _, child := range qn.children {
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
