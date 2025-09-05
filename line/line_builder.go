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
	uvs              []ebimath.Vector
}

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
	n := len(lb.Points)
	nSegments := n - 1
	isClosed := lb.Closed && n > 2
	if isClosed {
		nSegments = n
	}
	if lb.interpolateColor || interpolateWidth {
		for i := 0; i < nSegments; i++ {
			p1 := lb.Points[i%n]
			p2 := lb.Points[(i+1)%n]
			totalDistance += p1.DistanceTo(p2)
		}
	}

	// --- Initial state ---
	halfThickness := lb.Width / 2.0
	_ = halfThickness * halfThickness * lb.SharpLimit * lb.SharpLimit
	var currentDistance float64

	// --- Begin Cap ---
	if !isClosed && lb.BeginCapMode != LineCapNone {
		pA := lb.Points[0]
		pB := lb.Points[1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = lb.lerpWidth(0) * halfThickness
		}
		color := lb.lerpColor(0)

		v1 := pA.Add(normal.ScaleF(width))
		v2 := pA.Sub(normal.ScaleF(width))

		switch lb.BeginCapMode {
		case LineCapBox:
			v1_box := v1.Sub(dir.ScaleF(width))
			v2_box := v2.Sub(dir.ScaleF(width))
			lb.addTriangle(v1, v2, v1_box, color, color, color)
			lb.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			lb.newArc(pA, v1.Sub(pA), math.Pi, color, Rect{})
		}
	}

	// --- Main loop ---
	for i := 0; i < nSegments; i++ {
		paIndex := i % n
		pbIndex := (i + 1) % n
		pA := lb.Points[paIndex]
		pB := lb.Points[pbIndex]

		if pB.Equals(pA) {
			continue
		}

		dirAB := pB.Sub(pA)
		lenAB := dirAB.Length()
		if lenAB < 1e-6 {
			continue
		}

		tA := 0.0
		tB := 0.0
		if totalDistance > 0 {
			tA = currentDistance / totalDistance
			currentDistance += lenAB
			tB = currentDistance / totalDistance
		}

		colorA := lb.lerpColor(tA)
		colorB := lb.lerpColor(tB)

		widthA := halfThickness
		widthB := halfThickness
		if interpolateWidth {
			widthA = lb.lerpWidth(tA) * halfThickness
			widthB = lb.lerpWidth(tB) * halfThickness
		}

		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.Add(normal.ScaleF(widthA))
		vA2 := pA.Sub(normal.ScaleF(widthA))
		vB1 := pB.Add(normal.ScaleF(widthB))
		vB2 := pB.Sub(normal.ScaleF(widthB))

		lb.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		lb.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)

		// --- Joint ---
		if lb.JointMode != LineJointSharp {
			if !isClosed && i >= n-2 {
				continue
			}
			pcIndex := (i + 2) % n
			pC := lb.Points[pcIndex]
			if pC.Equals(pB) {
				continue
			}

			dirBC := pC.Sub(pB)
			if dirBC.Length() < 1e-6 {
				continue
			}

			normalB := dirBC.Normalize().Orthogonal()
			zCross := dirAB.Cross(dirBC)

			if math.Abs(zCross) > 1e-6 {
				vB1_next := pB.Add(normalB.ScaleF(widthB))
				vB2_next := pB.Sub(normalB.ScaleF(widthB))

				switch lb.JointMode {
				case LineJointBevel:
					if zCross < 0 {
						lb.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
					} else {
						lb.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
					}
				case LineJointRound:
					var vStart, vEnd ebimath.Vector
					if zCross < 0 {
						vStart, vEnd = vB1, vB1_next
					} else {
						vStart, vEnd = vB2, vB2_next
					}
					v1 := vStart.Sub(pB)
					v2 := vEnd.Sub(pB)
					angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
					lb.newArc(pB, v1, angle, colorB, Rect{})
				}
			}
		}
	}

	// --- End Cap ---
	if !isClosed && lb.EndCapMode != LineCapNone {
		pA := lb.Points[len(lb.Points)-2]
		pB := lb.Points[len(lb.Points)-1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = lb.lerpWidth(1.0) * halfThickness
		}
		color := lb.lerpColor(1.0)

		v1 := pB.Add(normal.ScaleF(width))
		v2 := pB.Sub(normal.ScaleF(width))

		switch lb.EndCapMode {
		case LineCapBox:
			v1_box := v1.Add(dir.ScaleF(width))
			v2_box := v2.Add(dir.ScaleF(width))
			lb.addTriangle(v1, v2, v1_box, color, color, color)
			lb.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			lb.newArc(pB, v1.Sub(pB), -math.Pi, color, Rect{})
		}
	}
}

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

func (lb *LineBuilder) lerpWidth(t float64) float64 {
	if len(lb.Widths) < 2 {
		return 1.0
	}
	widths := lb.Widths
	if lb.Closed {
		widths = append(widths, lb.Widths[0])
	}
	pos := t * float64(len(widths)-1)
	idx1 := int(pos)
	idx2 := idx1 + 1
	if idx2 >= len(widths) {
		return widths[len(widths)-1]
	}
	if idx1 < 0 {
		return widths[0]
	}
	localT := pos - float64(idx1)
	return widths[idx1] + (widths[idx2]-widths[idx1])*localT
}

func (lb *LineBuilder) newArc(center, vbegin ebimath.Vector, angleDelta float64, clr color.RGBA, uvRect Rect) {
	radius := vbegin.Length()
	angleStep := math.Pi / float64(lb.RoundPrecision)
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}

	t := vbegin.Angle()
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
		rpos := center.Add(ebimath.V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
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
