package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

type GridCornerJoinType int

const (
	GridCornerSharp GridCornerJoinType = iota
	GridCornerBevel
	GridCornerRound
)

// GridLine represents a grid of lines that can be rendered efficiently in a single draw call.
type GridLine struct {
	vertices            []ebiten.Vertex
	indices             []uint16
	whiteDot            *ebiten.Image
	position            ebimath.Vector
	size                int
	rows                int
	cols                int
	thickness           float64
	color               color.RGBA
	rotation            float64
	scale               ebimath.Vector
	isColorInterpolated bool
	topLeftColor        color.RGBA
	topRightColor       color.RGBA
	bottomLeftColor     color.RGBA
	bottomRightColor    color.RGBA
	cornerTL            GridCornerJoinType
	cornerTR            GridCornerJoinType
	cornerBL            GridCornerJoinType
	cornerBR            GridCornerJoinType
}

// NewGridLine creates a new grid with the specified dimensions and properties.
func NewGridLine(position ebimath.Vector, size, rows, cols int, thick float64) *GridLine {
	g := &GridLine{
		position:  position,
		size:      size,
		rows:      rows,
		cols:      cols,
		thickness: thick,
		color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		vertices:  make([]ebiten.Vertex, 0),
		indices:   make([]uint16, 0),
		whiteDot:  ebiten.NewImage(1, 1),
		rotation:  0,
		scale:     ebimath.V(1, 1),
		cornerTL:  GridCornerSharp,
		cornerTR:  GridCornerSharp,
		cornerBL:  GridCornerSharp,
		cornerBR:  GridCornerSharp,
	}
	g.whiteDot.Fill(color.White)
	g.buildMesh()
	return g
}

func (g *GridLine) buildMesh() {
	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	_ = g.thickness / 2.0
	totalWidth := float64(g.cols * g.size)
	totalHeight := float64(g.rows * g.size)
	center := ebimath.V(totalWidth/2, totalHeight/2)

	// Inner Horizontal lines
	for i := 1; i < g.rows; i++ {
		p1 := ebimath.Vector{X: 0, Y: float64(i * g.size)}
		p2 := ebimath.Vector{X: totalWidth, Y: float64(i * g.size)}
		g.addLine(p1, p2, center)
	}

	// Inner Vertical lines
	for i := 1; i < g.cols; i++ {
		p1 := ebimath.Vector{X: float64(i * g.size), Y: 0}
		p2 := ebimath.Vector{X: float64(i * g.size), Y: totalHeight}
		g.addLine(p1, p2, center)
	}

	// Outer lines
	// Top horizontal
	p1 := ebimath.V(0, 0)
	p2 := ebimath.V(totalWidth, 0)
	g.addLine(p1, p2, center)

	// Bottom horizontal
	p1 = ebimath.V(0, totalHeight)
	p2 = ebimath.V(totalWidth, totalHeight)
	g.addLine(p1, p2, center)

	// Left vertical
	p1 = ebimath.V(0, 0)
	p2 = ebimath.V(0, totalHeight)
	g.addLine(p1, p2, center)

	// Right vertical
	p1 = ebimath.V(totalWidth, 0)
	p2 = ebimath.V(totalWidth, totalHeight)
	g.addLine(p1, p2, center)

	// Add joints for corners
	corner := ebimath.V(0, 0)
	clr := g.getColor(corner)
	dirIn := ebimath.V(0, -1)
	dirOut := ebimath.V(1, 0)
	g.addJoint(corner, dirIn, dirOut, g.cornerTL, clr, center)

	corner = ebimath.V(totalWidth, 0)
	clr = g.getColor(corner)
	dirIn = ebimath.V(1, 0)
	dirOut = ebimath.V(0, 1)
	g.addJoint(corner, dirIn, dirOut, g.cornerTR, clr, center)

	corner = ebimath.V(totalWidth, totalHeight)
	clr = g.getColor(corner)
	dirIn = ebimath.V(0, 1)
	dirOut = ebimath.V(-1, 0)
	g.addJoint(corner, dirIn, dirOut, g.cornerBR, clr, center)

	corner = ebimath.V(0, totalHeight)
	clr = g.getColor(corner)
	dirIn = ebimath.V(-1, 0)
	dirOut = ebimath.V(0, -1)
	g.addJoint(corner, dirIn, dirOut, g.cornerBL, clr, center)
}

func (g *GridLine) addJoint(corner, dirIn, dirOut ebimath.Vector, joinType GridCornerJoinType, clr color.RGBA, cntr ebimath.Vector) {
	dirAB := dirIn.Normalize()
	dirBC := dirOut.Normalize()
	normal := dirAB.Orthogonal()
	normalB := dirBC.Orthogonal()
	widthB := g.thickness / 2.0

	vB1 := corner.Add(normal.ScaleF(widthB))
	vB2 := corner.Sub(normal.ScaleF(widthB))
	vB1_next := corner.Add(normalB.ScaleF(widthB))
	vB2_next := corner.Sub(normalB.ScaleF(widthB))

	zCross := dirAB.Cross(dirBC)

	var vStart, vEnd ebimath.Vector
	if zCross < 0 {
		vStart, vEnd = vB1, vB1_next
	} else {
		vStart, vEnd = vB2, vB2_next
	}

	switch joinType {
	case GridCornerSharp:
		var vB_outer, vB_outer_next ebimath.Vector
		if zCross < 0 {
			vB_outer, vB_outer_next = vB1, vB1_next
		} else {
			vB_outer, vB_outer_next = vB2, vB2_next
		}
		d := vB_outer_next.Sub(vB_outer)
		u := dirAB
		v := dirBC
		denom := u.Cross(v)
		if math.Abs(denom) < 1e-6 {
			return
		}
		t := d.Cross(v) / denom
		M := vB_outer.Add(u.ScaleF(t))
		g.addTriangleLocal(corner, vB_outer, M, clr, clr, clr, cntr)
		g.addTriangleLocal(corner, M, vB_outer_next, clr, clr, clr, cntr)
	case GridCornerBevel:
		g.addTriangleLocal(corner, vStart, vEnd, clr, clr, clr, cntr)
	case GridCornerRound:
		v1 := vStart.Sub(corner)
		v2 := vEnd.Sub(corner)
		angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
		g.newArc(corner, v1, angle, clr, cntr)
	}
}

func (g *GridLine) addTriangleLocal(v1, v2, v3 ebimath.Vector, c1, c2, c3 color.RGBA, center ebimath.Vector) {
	v1t := g.transformPos(v1, center)
	v2t := g.transformPos(v2, center)
	v3t := g.transformPos(v3, center)

	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255

	idx := uint16(len(g.vertices))

	g.vertices = append(g.vertices,
		ebiten.Vertex{DstX: float32(v1t.X), DstY: float32(v1t.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v2t.X), DstY: float32(v2t.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v3t.X), DstY: float32(v3t.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3, SrcX: 0, SrcY: 0},
	)
	g.indices = append(g.indices, idx, idx+1, idx+2)
}

func (g *GridLine) newArc(centerPos, vbegin ebimath.Vector, angleDelta float64, clr color.RGBA, center ebimath.Vector) {
	radius := vbegin.Length()
	angleStep := math.Pi / 8.0
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}

	t := vbegin.Angle()
	r, gcol, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(gcol)/255, float32(b)/255, float32(a)/255

	vi := len(g.vertices)
	centert := g.transformPos(centerPos, center)
	g.vertices = append(g.vertices, ebiten.Vertex{DstX: float32(centert.X), DstY: float32(centert.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0})

	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := centerPos.Add(ebimath.V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		rpost := g.transformPos(rpos, center)
		g.vertices = append(g.vertices, ebiten.Vertex{DstX: float32(rpost.X), DstY: float32(rpost.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0})
	}

	for i := 0; i < steps; i++ {
		g.indices = append(g.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func (g *GridLine) transformPos(pos, center ebimath.Vector) ebimath.Vector {
	scaledCenter := center.Scale(g.scale)
	return pos.Scale(g.scale).Sub(scaledCenter).Rotate(g.rotation).Add(scaledCenter).Add(g.position)
}

func (g *GridLine) addLine(p1, p2, center ebimath.Vector) {
	halfThick := g.thickness / 2.0
	dir := p2.Sub(p1)
	if dir.Length() == 0 {
		return
	}
	dir = dir.Normalize()
	normal := dir.Orthogonal()

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

	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255
	r4, g4, b4, a4 := c4.R, c4.G, c4.B, c4.A
	cr4, cg4, cb4, ca4 := float32(r4)/255, float32(g4)/255, float32(b4)/255, float32(a4)/255

	idx := uint16(len(g.vertices))

	g.vertices = append(g.vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: cr4, ColorG: cg4, ColorB: cb4, ColorA: ca4, SrcX: 0, SrcY: 0},
	)
	g.indices = append(g.indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

func (g *GridLine) getColor(pos ebimath.Vector) color.RGBA {
	if !g.isColorInterpolated {
		return g.color
	}
	totalWidth := float64(g.cols * g.size)
	totalHeight := float64(g.rows * g.size)
	return g.calculateInterpolatedColor(pos, totalWidth, totalHeight)
}

func (g *GridLine) calculateInterpolatedColor(pos ebimath.Vector, totalWidth, totalHeight float64) color.RGBA {
	tx := pos.X / totalWidth
	ty := pos.Y / totalHeight

	tx = math.Max(0, math.Min(1, tx))
	ty = math.Max(0, math.Min(1, ty))

	topColor := utils.LerpPremultipliedRGBA(g.topLeftColor, g.topRightColor, tx)
	bottomColor := utils.LerpPremultipliedRGBA(g.bottomLeftColor, g.bottomRightColor, tx)
	finalColor := utils.LerpPremultipliedRGBA(topColor, bottomColor, ty)
	return finalColor
}

// SetColor sets the color of the grid.
func (g *GridLine) SetColor(c color.RGBA) {
	g.color = c
	g.buildMesh()
}

// SetPosition sets the position of the grid.
func (g *GridLine) SetPosition(pos ebimath.Vector) {
	g.position = pos
	g.buildMesh()
}

// SetRotation sets the rotation of the grid.
func (g *GridLine) SetRotation(angle float64) {
	g.rotation = angle
	g.buildMesh()
}

// SetScale sets the scale of the grid.
func (g *GridLine) SetScale(scale ebimath.Vector) {
	g.scale = scale
	g.buildMesh()
}

// InterpolateColor sets the corner colors for gradient interpolation across the grid.
func (g *GridLine) InterpolateColor(topLeft, topRight, bottomLeft, bottomRight color.RGBA) {
	g.isColorInterpolated = true
	g.topLeftColor = topLeft
	g.topRightColor = topRight
	g.bottomLeftColor = bottomLeft
	g.bottomRightColor = bottomRight
	g.buildMesh()
}

// SetCornerJoin sets the join types for the four corners of the grid.
func (g *GridLine) SetCornerJoin(tl, tr, bl, br GridCornerJoinType) {
	g.cornerTL = tl
	g.cornerTR = tr
	g.cornerBL = bl
	g.cornerBR = br
	g.buildMesh()
}

// Draw renders the grid to the screen.
func (g *GridLine) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(g.vertices) == 0 {
		return
	}
	screen.DrawTriangles(g.vertices, g.indices, g.whiteDot, op)
}
