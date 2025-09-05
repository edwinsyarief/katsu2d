package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// Rect is a 2D rectangle defined by a position and a size.
type Rect struct {
	Position ebimath.Vector
	Size     ebimath.Vector
}

// LineBuilder is responsible for generating the vertices and indices for a line.
type LineBuilder struct {
	Points           []ebimath.Vector
	JointMode        LineJointMode
	BeginCapMode     LineCapMode
	EndCapMode       LineCapMode
	Closed           bool
	Width            float64
	Widths           []float64
	DefaultColor     color.RGBA
	Colors           []color.RGBA
	TextureMode      LineTextureMode
	SharpLimit       float64
	RoundPrecision   int
	TileAspect       float64
	Vertices         []ebiten.Vertex
	Indices          []uint16
	interpolateColor bool
	lastIndex        [2]int
	uvs              []ebimath.Vector
}

type orientation int

const (
	up orientation = iota
	down
)

func NewLineBuilder() *LineBuilder {
	return &LineBuilder{
		SharpLimit:     2.0,
		RoundPrecision: 8,
		TileAspect:     1.0,
		Width:          10.0,
		DefaultColor:   color.RGBA{R: 102, G: 128, B: 255, A: 255},
	}
}

func (lb *LineBuilder) Build() {
	if len(lb.Points) < 2 {
		return // Not enough points to draw a line
	}

	// --- Pre-computation for interpolation ---
	lb.interpolateColor = len(lb.Colors) > 1
	interpolateWidth := len(lb.Widths) > 1
	var totalDistance float64
	if lb.interpolateColor || interpolateWidth {
		for i := 1; i < len(lb.Points); i++ {
			totalDistance += vecDistanceTo(lb.Points[i], lb.Points[i-1])
		}
	}

	// --- Initial state ---
	halfThickness := lb.Width / 2.0
	sqMiterThreshold := halfThickness * halfThickness * lb.SharpLimit * lb.SharpLimit
	var currentDistance float64
	isClosed := lb.Closed && len(lb.Points) > 2

	// --- Begin Cap ---
	if !isClosed {
		pA := lb.Points[0]
		pB := lb.Points[1]
		dir := vecNormalized(vecSub(pB, pA))
		normal := vecOrthogonal(dir)
		width := halfThickness
		if interpolateWidth {
			width = lb.lerpWidth(0) * halfThickness
		}
		color := lb.lerpColor(0)

		v1 := vecAdd(pA, vecMul(normal, width))
		v2 := vecSub(pA, vecMul(normal, width))

		switch lb.BeginCapMode {
		case LineCapBox:
			v1 = vecSub(v1, vecMul(dir, width))
			v2 = vecSub(v2, vecMul(dir, width))
			lb.addTriangle(v1, v2, pA, color, color, color)
		case LineCapRound:
			lb.newArc(pA, vecSub(v1, pA), -math.Pi, color, Rect{})
		}
	}

	// --- Main loop ---
	var pA, pB, pC ebimath.Vector
	pA = lb.Points[0]

	idxB := 1
	for idxB < len(lb.Points) && lb.Points[idxB].Equals(pA) {
		idxB++
	}
	if idxB >= len(lb.Points) {
		return
	}
	pB = lb.Points[idxB]

	for i := idxB; i <= len(lb.Points); i++ {
		if i < len(lb.Points) {
			pC = lb.Points[i]
			if pC.Equals(pB) {
				continue
			}
		} else {
			if isClosed {
				break
			} // Handled by joint logic
			pC = vecAdd(pB, vecSub(pB, pA))
		}

		dirAB := vecSub(pB, pA)
		lenAB := vecLength(dirAB)
		if lenAB < 1e-6 {
			pA = pB
			pB = pC
			continue
		}

		tA := currentDistance / totalDistance
		currentDistance += lenAB
		tB := currentDistance / totalDistance

		colorA := lb.lerpColor(tA)
		colorB := lb.lerpColor(tB)

		widthA := halfThickness
		widthB := halfThickness
		if interpolateWidth {
			widthA = lb.lerpWidth(tA) * halfThickness
			widthB = lb.lerpWidth(tB) * halfThickness
		}

		normalA := vecOrthogonal(vecNormalized(dirAB))
		vA1 := vecAdd(pA, vecMul(normalA, widthA))
		vA2 := vecSub(pA, vecMul(normalA, widthA))
		vB1 := vecAdd(pB, vecMul(normalA, widthB))
		vB2 := vecSub(pB, vecMul(normalA, widthB))

		lb.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		lb.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)

		if i >= len(lb.Points) {
			break
		}
		if isClosed && i == len(lb.Points)-1 {
			pC = lb.Points[0]
		}

		dirBC := vecSub(pC, pB)
		lenBC := vecLength(dirBC)
		if lenBC < 1e-6 {
			continue
		}

		normalB := vecOrthogonal(vecNormalized(dirBC))
		zCross := vecCross(dirAB, dirBC)

		if math.Abs(zCross) > 1e-6 {
			vB1_next := vecAdd(pB, vecMul(normalB, widthB))
			vB2_next := vecSub(pB, vecMul(normalB, widthB))

			switch lb.JointMode {
			case LineJointBevel:
				if zCross < 0 { // Right turn
					lb.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
				} else { // Left turn
					lb.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
				}

			case LineJointSharp:
				// Bevel part
				if zCross < 0 {
					lb.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
				} else {
					lb.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
				}
				// Miter part
				var miterPoint ebimath.Vector
				var success bool
				if zCross < 0 {
					miterPoint, success = segmentIntersectsSegment(vA1, vB1, vB1_next, vecAdd(pC, vecMul(normalB, widthB)))
				} else {
					miterPoint, success = segmentIntersectsSegment(vA2, vB2, vB2_next, vecSub(pC, vecMul(normalB, widthB)))
				}
				if success && vecDistanceSquaredTo(miterPoint, pB) <= sqMiterThreshold {
					if zCross < 0 {
						lb.addTriangle(vB1, miterPoint, vB1_next, colorB, colorB, colorB)
					} else {
						lb.addTriangle(vB2, miterPoint, vB2_next, colorB, colorB, colorB)
					}
				}

			case LineJointRound:
				var vStart, vEnd ebimath.Vector
				if zCross < 0 { // Right turn
					vStart = vB1
					vEnd = vB1_next
				} else { // Left turn
					vStart = vB2
					vEnd = vB2_next
				}
				v1 := vecSub(vStart, pB)
				v2 := vecSub(vEnd, pB)
				angle := vecAngleTo(v1, v2)
				lb.newArc(pB, v1, angle, colorB, Rect{})
			}
		}
		pA = pB
		pB = pC
	}

	// --- End Cap ---
	if !isClosed {
		pA := lb.Points[len(lb.Points)-2]
		pB := lb.Points[len(lb.Points)-1]
		dir := vecNormalized(vecSub(pB, pA))
		normal := vecOrthogonal(dir)
		width := halfThickness
		if interpolateWidth {
			width = lb.lerpWidth(1.0) * halfThickness
		}
		color := lb.lerpColor(1.0)

		v1 := vecAdd(pB, vecMul(normal, width))
		v2 := vecSub(pB, vecMul(normal, width))

		switch lb.EndCapMode {
		case LineCapBox:
			v1 = vecAdd(v1, vecMul(dir, width))
			v2 = vecAdd(v2, vecMul(dir, width))
			lb.addTriangle(v1, v2, pB, color, color, color)
		case LineCapRound:
			lb.newArc(pB, vecSub(v1, pB), math.Pi, color, Rect{})
		}
	}
}

// addTriangle is a helper to add a triangle to the mesh.
func (lb *LineBuilder) addTriangle(v1, v2, v3 ebimath.Vector, c1, c2, c3 color.RGBA) {
	idx := uint16(len(lb.Vertices))

	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255

	lb.Vertices = append(lb.Vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3},
	)
	lb.Indices = append(lb.Indices, idx, idx+1, idx+2)
}

// lerpWidth performs linear interpolation on the Widths slice.
func (lb *LineBuilder) lerpWidth(t float64) float64 {
	if len(lb.Widths) < 2 {
		return 1.0
	}
	pos := t * float64(len(lb.Widths)-1)
	idx1 := int(pos)
	idx2 := idx1 + 1
	if idx2 >= len(lb.Widths) {
		return lb.Widths[len(lb.Widths)-1]
	}
	if idx1 < 0 {
		return lb.Widths[0]
	}
	localT := pos - float64(idx1)
	return lb.Widths[idx1] + (lb.Widths[idx2]-lb.Widths[idx1])*localT
}

func (lb *LineBuilder) stripBegin(pUp, pDown ebimath.Vector, clr color.RGBA, uvx float64) {
	vi := len(lb.Vertices)
	r, g, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255

	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(pUp.X), DstY: float32(pUp.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(pDown.X), DstY: float32(pDown.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})

	if lb.TextureMode != LineTextureNone {
		lb.uvs = append(lb.uvs, ebimath.V(uvx, 0.0), ebimath.V(uvx, 1.0))
	}

	lb.lastIndex[up] = vi
	lb.lastIndex[down] = vi + 1
}

func (lb *LineBuilder) stripAddQuad(pUp, pDown ebimath.Vector, clr color.RGBA, uvx float64) {
	vi := len(lb.Vertices)
	r, g, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255

	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(pUp.X), DstY: float32(pUp.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(pDown.X), DstY: float32(pDown.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})

	if lb.TextureMode != LineTextureNone {
		lb.uvs = append(lb.uvs, ebimath.V(uvx, 0.0), ebimath.V(uvx, 1.0))
	}

	lb.Indices = append(lb.Indices, uint16(lb.lastIndex[up]), uint16(vi+1), uint16(lb.lastIndex[down]), uint16(lb.lastIndex[up]), uint16(vi), uint16(vi+1))
	lb.lastIndex[up] = vi
	lb.lastIndex[down] = vi + 1
}

func (lb *LineBuilder) stripAddTri(pUp ebimath.Vector, o orientation) {
	vi := len(lb.Vertices)
	v := lb.Vertices[len(lb.Vertices)-1]
	cr, cg, cb, ca := v.ColorR, v.ColorG, v.ColorB, v.ColorA

	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(pUp.X), DstY: float32(pUp.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})

	oppositeO := up
	if o == up {
		oppositeO = down
	}

	if lb.TextureMode != LineTextureNone {
		lb.uvs = append(lb.uvs, lb.uvs[lb.lastIndex[oppositeO]])
	}

	lb.Indices = append(lb.Indices, uint16(lb.lastIndex[oppositeO]), uint16(vi), uint16(lb.lastIndex[o]))
	lb.lastIndex[oppositeO] = vi
}

func (lb *LineBuilder) stripAddArc(center ebimath.Vector, angleDelta float64, o orientation) {
	oppositeO := up
	if o == up {
		oppositeO = down
	}
	vbegin := vecSub(ebimath.V(float64(lb.Vertices[lb.lastIndex[oppositeO]].DstX), float64(lb.Vertices[lb.lastIndex[oppositeO]].DstY)), center)
	radius := vecLength(vbegin)
	angleStep := math.Pi / float64(lb.RoundPrecision)
	steps := int(math.Abs(angleDelta) / angleStep)

	if angleDelta < 0 {
		angleStep = -angleStep
	}
	t := math.Atan2(vbegin.Y, vbegin.X)
	for i := 0; i < steps; i++ {
		t += angleStep
		rpos := vecAdd(center, vecMul(ebimath.V(math.Cos(t), math.Sin(t)), radius))
		lb.stripAddTri(rpos, o)
	}
	endAngle := math.Atan2(vbegin.Y, vbegin.X) + angleDelta
	rpos := vecAdd(center, vecMul(ebimath.V(math.Cos(endAngle), math.Sin(endAngle)), radius))
	lb.stripAddTri(rpos, o)
}

func (lb *LineBuilder) newArc(center, vbegin ebimath.Vector, angleDelta float64, clr color.RGBA, uvRect Rect) {
	radius := vecLength(vbegin)
	angleStep := math.Pi / float64(lb.RoundPrecision)
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}

	t := math.Atan2(vbegin.Y, vbegin.X)
	r, g, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255

	vi := len(lb.Vertices)
	lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(center.X), DstY: float32(center.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	if lb.TextureMode != LineTextureNone {
		lb.uvs = append(lb.uvs, interpolate(uvRect, ebimath.V(0.5, 0.5)))
	}

	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := vecAdd(center, vecMul(ebimath.V(math.Cos(angle), math.Sin(angle)), radius))
		lb.Vertices = append(lb.Vertices, ebiten.Vertex{DstX: float32(rpos.X), DstY: float32(rpos.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	}

	for i := 0; i < steps; i++ {
		lb.Indices = append(lb.Indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func (lb *LineBuilder) lerpColor(t float64) color.RGBA {
	if !lb.interpolateColor || len(lb.Colors) == 0 {
		if len(lb.Colors) == 1 {
			return lb.Colors[0]
		}
		return lb.DefaultColor
	}
	if len(lb.Colors) == 1 {
		return lb.Colors[0]
	}

	colors := lb.Colors
	if lb.Closed {
		colors = append(colors, lb.Colors[0])
	}

	pos := t * float64(len(colors)-1)
	segment := int(pos)
	tInSegment := pos - float64(segment)

	if segment >= len(colors)-1 {
		segment = len(colors) - 2
		tInSegment = 1.0
	}
	if segment < 0 {
		segment = 0
		tInSegment = 0.0
	}

	return utils.LerpPremultipliedRGBA(colors[segment], colors[segment+1], tInSegment)
}

func interpolate(r Rect, v ebimath.Vector) ebimath.Vector {
	return ebimath.V(
		r.Position.X+(r.Position.X+r.Size.X-r.Position.X)*v.X,
		r.Position.Y+(r.Position.Y+r.Size.Y-r.Position.Y)*v.Y,
	)
}

func segmentIntersectsSegment(p1, p2, q1, q2 ebimath.Vector) (ebimath.Vector, bool) {
	d := (p2.X-p1.X)*(q2.Y-q1.Y) - (p2.Y-p1.Y)*(q2.X-q1.X)
	if math.Abs(d) < 1e-6 {
		return ebimath.Vector{}, false
	}
	t := ((q1.X-p1.X)*(q2.Y-q1.Y) - (q1.Y-p1.Y)*(q2.X-q1.X)) / d
	u := -((p2.X-p1.X)*(q1.Y-p1.Y) - (p2.Y-p1.Y)*(q1.X-p1.X)) / d
	if t >= 0 && t <= 1 && u >= 0 && u <= 1 {
		return ebimath.V(p1.X+t*(p2.X-p1.X), p1.Y+t*(p2.Y-p1.Y)), true
	}
	return ebimath.Vector{}, false
}

func vecLength(v ebimath.Vector) float64                { return math.Sqrt(v.X*v.X + v.Y*v.Y) }
func vecSub(v1, v2 ebimath.Vector) ebimath.Vector       { return ebimath.V(v1.X-v2.X, v1.Y-v2.Y) }
func vecAdd(v1, v2 ebimath.Vector) ebimath.Vector       { return ebimath.V(v1.X+v2.X, v1.Y+v2.Y) }
func vecMul(v ebimath.Vector, s float64) ebimath.Vector { return ebimath.V(v.X*s, v.Y*s) }
func vecNormalized(v ebimath.Vector) ebimath.Vector {
	l := vecLength(v)
	if l == 0 {
		return ebimath.Vector{}
	}
	return ebimath.V(v.X/l, v.Y/l)
}
func vecOrthogonal(v ebimath.Vector) ebimath.Vector { return ebimath.V(-v.Y, v.X) }
func vecDot(v1, v2 ebimath.Vector) float64          { return v1.X*v2.X + v1.Y*v2.Y }
func vecCross(v1, v2 ebimath.Vector) float64        { return v1.X*v2.Y - v1.Y*v2.X }
func vecDistanceTo(v1, v2 ebimath.Vector) float64   { return vecLength(vecSub(v1, v2)) }
func vecDistanceSquaredTo(v1, v2 ebimath.Vector) float64 {
	dx, dy := v1.X-v2.X, v1.Y-v2.Y
	return dx*dx + dy*dy
}
func vecLerp(v1, v2 ebimath.Vector, t float64) ebimath.Vector {
	return vecAdd(v1, vecMul(vecSub(v2, v1), t))
}
func vecAngleTo(v1, v2 ebimath.Vector) float64 {
	return math.Atan2(v1.X*v2.Y-v1.Y*v2.X, v1.X*v2.X+v1.Y*v2.Y)
}
