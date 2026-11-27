package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// GridCornerJoinType defines the type of join to use for the corners of a grid.
// This allows for different visual styles like sharp corners, beveled edges, or rounded joints.
type GridCornerJoinType int

const (
	// GridCornerSharp creates a sharp, pointed corner where two lines meet.
	GridCornerSharp GridCornerJoinType = iota
	// GridCornerBevel creates a beveled or flat corner, effectively cutting off the sharp point.
	GridCornerBevel
	// GridCornerRound creates a smooth, rounded corner using an arc.
	GridCornerRound
)

// GridLine represents a grid of lines that can be rendered efficiently in a single draw call.
// The grid is built as a mesh of triangles, which can be transformed and drawn as a single unit.
type GridLine struct {
	whiteDot            *ebiten.Image
	vertices            []ebiten.Vertex
	indices             []uint16
	position            Vector
	scale               Vector
	size                int
	rows                int
	cols                int
	thickness           float64
	rotation            float64
	cornerTL            GridCornerJoinType
	cornerTR            GridCornerJoinType
	cornerBL            GridCornerJoinType
	cornerBR            GridCornerJoinType
	color               color.RGBA
	topLeftColor        color.RGBA
	topRightColor       color.RGBA
	bottomLeftColor     color.RGBA
	bottomRightColor    color.RGBA
	isColorInterpolated bool
}

// NewGridLine creates a new grid with the specified dimensions and properties.
// It initializes the object and builds the initial mesh geometry.
func NewGridLine(position Vector, size, rows, cols int, thick float64) *GridLine {
	g := &GridLine{
		position:  position,
		size:      size,
		rows:      rows,
		cols:      cols,
		thickness: thick,
		color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		whiteDot:  defaultWhiteImage(),
		rotation:  0,
		scale:     V(1, 1),
		cornerTL:  GridCornerSharp,
		cornerTR:  GridCornerSharp,
		cornerBL:  GridCornerSharp,
		cornerBR:  GridCornerSharp,
	}
	g.buildMesh()
	return g
}

func (g *GridLine) buildMesh() {
	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	totalWidth := float64(g.cols * g.size)
	totalHeight := float64(g.rows * g.size)
	center := V(totalWidth/2, totalHeight/2)

	// Inner horizontal lines.
	for i := 1; i < g.rows; i++ {
		y := float64(i * g.size)
		g.addLine(Vector{X: 0, Y: y}, Vector{X: totalWidth, Y: y}, center)
	}
	// Inner vertical lines.
	for i := 1; i < g.cols; i++ {
		x := float64(i * g.size)
		g.addLine(Vector{X: x, Y: 0}, Vector{X: x, Y: totalHeight}, center)
	}
	// Outer boundary lines.
	g.addLine(V(0, 0), V(totalWidth, 0), center)
	g.addLine(V(0, totalHeight), V(totalWidth, totalHeight), center)
	g.addLine(V(0, 0), V(0, totalHeight), center)
	g.addLine(V(totalWidth, 0), V(totalWidth, totalHeight), center)

	// Corner joints.
	corner := V(0, 0)
	g.addJoint(corner, V(0, -1), V(1, 0), g.cornerTL, g.getColor(corner), center)

	corner = V(totalWidth, 0)
	g.addJoint(corner, V(1, 0), V(0, 1), g.cornerTR, g.getColor(corner), center)

	corner = V(totalWidth, totalHeight)
	g.addJoint(corner, V(0, 1), V(-1, 0), g.cornerBR, g.getColor(corner), center)

	corner = V(0, totalHeight)
	g.addJoint(corner, V(-1, 0), V(0, -1), g.cornerBL, g.getColor(corner), center)
}

func (g *GridLine) addJoint(corner, dirIn, dirOut Vector, joinType GridCornerJoinType, clr color.RGBA, cntr Vector) {
	dirAB := dirIn.Normalize()
	dirBC := dirOut.Normalize()
	normal := dirAB.Orthogonal()
	normalB := dirBC.Orthogonal()
	widthB := g.thickness / 2.0

	vB1 := corner.Add(normal.ScaleF(widthB))
	vB2 := corner.Sub(normal.ScaleF(widthB))
	vB1Next := corner.Add(normalB.ScaleF(widthB))
	vB2Next := corner.Sub(normalB.ScaleF(widthB))

	zCross := dirAB.Cross(dirBC)
	vStart, vEnd := vB2, vB2Next
	if zCross < 0 {
		vStart, vEnd = vB1, vB1Next
	}

	switch joinType {
	case GridCornerSharp:
		d := vEnd.Sub(vStart)
		denom := dirAB.Cross(dirBC)
		if math.Abs(denom) < 1e-6 {
			return
		}
		t := d.Cross(dirBC) / denom
		miter := vStart.Add(dirAB.ScaleF(t))
		g.addTriangleLocal(corner, vStart, miter, clr, clr, clr, cntr)
		g.addTriangleLocal(corner, miter, vEnd, clr, clr, clr, cntr)
	case GridCornerBevel:
		g.addTriangleLocal(corner, vStart, vEnd, clr, clr, clr, cntr)
	case GridCornerRound:
		v1 := vStart.Sub(corner)
		v2 := vEnd.Sub(corner)
		angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
		g.newArc(corner, v1, angle, clr, cntr)
	}
}

func (g *GridLine) addTriangleLocal(v1, v2, v3 Vector, c1, c2, c3 color.RGBA, center Vector) {
	const inv255 = float32(1.0 / 255.0)
	v1t := g.transformPos(v1, center)
	v2t := g.transformPos(v2, center)
	v3t := g.transformPos(v3, center)

	idx := uint16(len(g.vertices))
	g.vertices = append(g.vertices,
		ebiten.Vertex{
			DstX: float32(v1t.X), DstY: float32(v1t.Y),
			ColorR: float32(c1.R) * inv255, ColorG: float32(c1.G) * inv255, ColorB: float32(c1.B) * inv255, ColorA: float32(c1.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v2t.X), DstY: float32(v2t.Y),
			ColorR: float32(c2.R) * inv255, ColorG: float32(c2.G) * inv255, ColorB: float32(c2.B) * inv255, ColorA: float32(c2.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v3t.X), DstY: float32(v3t.Y),
			ColorR: float32(c3.R) * inv255, ColorG: float32(c3.G) * inv255, ColorB: float32(c3.B) * inv255, ColorA: float32(c3.A) * inv255,
		},
	)
	g.indices = append(g.indices, idx, idx+1, idx+2)
}

func (g *GridLine) newArc(centerPos, vbegin Vector, angleDelta float64, clr color.RGBA, center Vector) {
	const inv255 = float32(1.0 / 255.0)
	radius := vbegin.Length()
	angleStep := math.Pi / 8.0
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}
	t := vbegin.Angle()
	cr := float32(clr.R) * inv255
	cg := float32(clr.G) * inv255
	cb := float32(clr.B) * inv255
	ca := float32(clr.A) * inv255

	vi := len(g.vertices)
	centert := g.transformPos(centerPos, center)
	g.vertices = append(g.vertices, ebiten.Vertex{
		DstX: float32(centert.X), DstY: float32(centert.Y),
		ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
	})
	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := centerPos.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		rpost := g.transformPos(rpos, center)
		g.vertices = append(g.vertices, ebiten.Vertex{
			DstX: float32(rpost.X), DstY: float32(rpost.Y),
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})
	}
	for i := range steps {
		g.indices = append(g.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func (g *GridLine) transformPos(pos, center Vector) Vector {
	scaledCenter := center.Scale(g.scale)
	return pos.Scale(g.scale).Sub(scaledCenter).Rotate(g.rotation).Add(scaledCenter).Add(g.position)
}

func (g *GridLine) addLine(p1, p2, center Vector) {
	const inv255 = float32(1.0 / 255.0)
	halfThick := g.thickness / 2.0
	dir := p2.Sub(p1)
	if dir.Length() == 0 {
		return
	}
	normal := dir.Normalize().Orthogonal()

	lv1 := p1.Add(normal.ScaleF(halfThick))
	lv2 := p1.Sub(normal.ScaleF(halfThick))
	lv3 := p2.Add(normal.ScaleF(halfThick))
	lv4 := p2.Sub(normal.ScaleF(halfThick))

	v1 := g.transformPos(lv1, center)
	v2 := g.transformPos(lv2, center)
	v3 := g.transformPos(lv3, center)
	v4 := g.transformPos(lv4, center)

	var c1, c2, c3, c4 color.RGBA
	if g.isColorInterpolated {
		totalWidth := float64(g.cols * g.size)
		totalHeight := float64(g.rows * g.size)
		c1 = g.calculateInterpolatedColor(lv1, totalWidth, totalHeight)
		c2 = g.calculateInterpolatedColor(lv2, totalWidth, totalHeight)
		c3 = g.calculateInterpolatedColor(lv3, totalWidth, totalHeight)
		c4 = g.calculateInterpolatedColor(lv4, totalWidth, totalHeight)
	} else {
		c1, c2, c3, c4 = g.color, g.color, g.color, g.color
	}

	idx := uint16(len(g.vertices))
	g.vertices = append(g.vertices,
		ebiten.Vertex{
			DstX: float32(v1.X), DstY: float32(v1.Y),
			ColorR: float32(c1.R) * inv255, ColorG: float32(c1.G) * inv255, ColorB: float32(c1.B) * inv255, ColorA: float32(c1.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v2.X), DstY: float32(v2.Y),
			ColorR: float32(c2.R) * inv255, ColorG: float32(c2.G) * inv255, ColorB: float32(c2.B) * inv255, ColorA: float32(c2.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v3.X), DstY: float32(v3.Y),
			ColorR: float32(c3.R) * inv255, ColorG: float32(c3.G) * inv255, ColorB: float32(c3.B) * inv255, ColorA: float32(c3.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v4.X), DstY: float32(v4.Y),
			ColorR: float32(c4.R) * inv255, ColorG: float32(c4.G) * inv255, ColorB: float32(c4.B) * inv255, ColorA: float32(c4.A) * inv255,
		},
	)
	g.indices = append(g.indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

func (g *GridLine) getColor(pos Vector) color.RGBA {
	if !g.isColorInterpolated {
		return g.color
	}
	totalWidth := float64(g.cols * g.size)
	totalHeight := float64(g.rows * g.size)
	return g.calculateInterpolatedColor(pos, totalWidth, totalHeight)
}

func (g *GridLine) calculateInterpolatedColor(pos Vector, totalWidth, totalHeight float64) color.RGBA {
	var tx, ty float64
	if totalWidth > 0 {
		tx = Clamp(pos.X/totalWidth, 0.0, 1.0)
	}
	if totalHeight > 0 {
		ty = Clamp(pos.Y/totalHeight, 0.0, 1.0)
	}
	topColor := LerpPremultipliedRGBA(g.topLeftColor, g.topRightColor, tx)
	bottomColor := LerpPremultipliedRGBA(g.bottomLeftColor, g.bottomRightColor, tx)
	return LerpPremultipliedRGBA(topColor, bottomColor, ty)
}

// SetColor sets the single, uniform color for the grid and rebuilds the mesh.
func (g *GridLine) SetColor(c color.RGBA) {
	g.color = c
	g.buildMesh()
}

// SetPosition sets the top-left position of the grid and rebuilds the mesh.
func (g *GridLine) SetPosition(pos Vector) {
	g.position = pos
	g.buildMesh()
}

// SetRotation sets the rotation angle for the grid in radians and rebuilds the mesh.
func (g *GridLine) SetRotation(angle float64) {
	g.rotation = angle
	g.buildMesh()
}

// SetScale sets the scaling factor for the grid along the X and Y axes and rebuilds the mesh.
func (g *GridLine) SetScale(scale Vector) {
	g.scale = scale
	g.buildMesh()
}

// InterpolateColor sets the corner colors for gradient interpolation across the grid and rebuilds the mesh.
func (g *GridLine) InterpolateColor(topLeft, topRight, bottomLeft, bottomRight color.RGBA) {
	g.isColorInterpolated = true
	g.topLeftColor = topLeft
	g.topRightColor = topRight
	g.bottomLeftColor = bottomLeft
	g.bottomRightColor = bottomRight
	g.buildMesh()
}

// SetCornerJoin sets the join types for the four corners of the grid and rebuilds the mesh.
func (g *GridLine) SetCornerJoin(tl, tr, bl, br GridCornerJoinType) {
	g.cornerTL = tl
	g.cornerTR = tr
	g.cornerBL = bl
	g.cornerBR = br
	g.buildMesh()
}

// Draw renders the grid's mesh to the screen using a single DrawTriangles call.
func (g *GridLine) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(g.vertices) == 0 {
		return
	}
	if g.whiteDot == nil {
		g.whiteDot = defaultWhiteImage()
	}
	screen.DrawTriangles(g.vertices, g.indices, g.whiteDot, op)
}
