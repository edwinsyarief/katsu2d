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
	points           []ebimath.Vector
	jointMode        LineJointMode
	beginCapMode     LineCapMode
	endCapMode       LineCapMode
	closed           bool
	width            float64
	widths           []float64
	defaultColor     color.RGBA
	colors           []color.RGBA
	textureMode      LineTextureMode
	sharpLimit       float64
	roundPrecision   int
	tileAspect       float64
	vertices         []ebiten.Vertex
	indices          []uint16
	interpolateColor bool
	uvs              []ebimath.Vector
}

func NewLineBuilder() *LineBuilder {
	return &LineBuilder{
		sharpLimit:     2.0,
		roundPrecision: 8,
		tileAspect:     1.0,
		width:          10.0,
		defaultColor:   color.RGBA{R: 102, G: 128, B: 255, A: 255},
	}
}

func (self *LineBuilder) Build() {
	if len(self.points) < 2 {
		return // Not enough points to draw a line
	}

	// --- Pre-computation for interpolation ---
	self.interpolateColor = len(self.colors) > 1
	interpolateWidth := len(self.widths) > 1
	var totalDistance float64
	n := len(self.points)
	nSegments := n - 1
	isClosed := self.closed && n > 2
	if isClosed {
		nSegments = n
	}
	if self.interpolateColor || interpolateWidth {
		for i := 0; i < nSegments; i++ {
			p1 := self.points[i%n]
			p2 := self.points[(i+1)%n]
			totalDistance += p1.DistanceTo(p2)
		}
	}

	// --- Initial state ---
	halfThickness := self.width / 2.0
	_ = halfThickness * halfThickness * self.sharpLimit * self.sharpLimit
	var currentDistance float64

	// --- Begin Cap ---
	if !isClosed && self.beginCapMode != LineCapNone {
		pA := self.points[0]
		pB := self.points[1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = self.lerpWidth(0) * halfThickness
		}
		color := self.lerpColor(0)

		v1 := pA.Add(normal.ScaleF(width))
		v2 := pA.Sub(normal.ScaleF(width))

		switch self.beginCapMode {
		case LineCapBox:
			v1_box := v1.Sub(dir.ScaleF(width))
			v2_box := v2.Sub(dir.ScaleF(width))
			self.addTriangle(v1, v2, v1_box, color, color, color)
			self.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			self.newArc(pA, v1.Sub(pA), math.Pi, color, Rect{})
		}
	}

	// --- Main loop ---
	for i := 0; i < nSegments; i++ {
		paIndex := i % n
		pbIndex := (i + 1) % n
		pA := self.points[paIndex]
		pB := self.points[pbIndex]

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

		colorA := self.lerpColor(tA)
		colorB := self.lerpColor(tB)

		widthA := halfThickness
		widthB := halfThickness
		if interpolateWidth {
			widthA = self.lerpWidth(tA) * halfThickness
			widthB = self.lerpWidth(tB) * halfThickness
		}

		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.Add(normal.ScaleF(widthA))
		vA2 := pA.Sub(normal.ScaleF(widthA))
		vB1 := pB.Add(normal.ScaleF(widthB))
		vB2 := pB.Sub(normal.ScaleF(widthB))

		self.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		self.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)

		// --- Joint ---
		if self.jointMode != LineJointSharp {
			if !isClosed && i >= n-2 {
				continue
			}
			pcIndex := (i + 2) % n
			pC := self.points[pcIndex]
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

				switch self.jointMode {
				case LineJointBevel:
					if zCross < 0 {
						self.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
					} else {
						self.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
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
					self.newArc(pB, v1, angle, colorB, Rect{})
				}
			}
		}
	}

	// --- End Cap ---
	if !isClosed && self.endCapMode != LineCapNone {
		pA := self.points[len(self.points)-2]
		pB := self.points[len(self.points)-1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = self.lerpWidth(1.0) * halfThickness
		}
		color := self.lerpColor(1.0)

		v1 := pB.Add(normal.ScaleF(width))
		v2 := pB.Sub(normal.ScaleF(width))

		switch self.endCapMode {
		case LineCapBox:
			v1_box := v1.Add(dir.ScaleF(width))
			v2_box := v2.Add(dir.ScaleF(width))
			self.addTriangle(v1, v2, v1_box, color, color, color)
			self.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			self.newArc(pB, v1.Sub(pB), -math.Pi, color, Rect{})
		}
	}
}

func (self *LineBuilder) addTriangle(v1, v2, v3 ebimath.Vector, c1, c2, c3 color.RGBA) {
	idx := uint16(len(self.vertices))

	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255

	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3},
	)
	self.indices = append(self.indices, idx, idx+1, idx+2)
}

func (self *LineBuilder) lerpWidth(t float64) float64 {
	if len(self.widths) < 2 {
		return 1.0
	}
	widths := self.widths
	if self.closed {
		widths = append(widths, self.widths[0])
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

func (self *LineBuilder) newArc(center, vbegin ebimath.Vector, angleDelta float64, clr color.RGBA, uvRect Rect) {
	radius := vbegin.Length()
	angleStep := math.Pi / float64(self.roundPrecision)
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}

	t := vbegin.Angle()
	r, g, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(g)/255, float32(b)/255, float32(a)/255

	vi := len(self.vertices)
	self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(center.X), DstY: float32(center.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	if self.textureMode != LineTextureNone {
		self.uvs = append(self.uvs, interpolate(uvRect, ebimath.V(0.5, 0.5)))
	}

	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := center.Add(ebimath.V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(rpos.X), DstY: float32(rpos.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	}

	for i := 0; i < steps; i++ {
		self.indices = append(self.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func (self *LineBuilder) lerpColor(t float64) color.RGBA {
	if !self.interpolateColor || len(self.colors) == 0 {
		if len(self.colors) == 1 {
			return self.colors[0]
		}
		return self.defaultColor
	}
	if len(self.colors) == 1 {
		return self.colors[0]
	}

	colors := self.colors
	if self.closed {
		colors = append(colors, self.colors[0])
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
