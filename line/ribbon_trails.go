package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// trailPoint holds information about a single point in the ribbon trail.
type trailPoint struct {
	pos          ebimath.Vector
	creationTime float64
}

// RibbonTrails creates a trail effect using a line that fades and scales over time.
type RibbonTrails struct {
	points            []*trailPoint
	SegmentLifetime   float64
	SegmentWidths     []float64
	SegmentColors     []color.RGBA
	TotalSegmentLimit int
	currentTime       float64

	// Mesh generation properties, similar to LineBuilder
	vertices       []ebiten.Vertex
	indices        []uint16
	jointMode      LineJointMode
	roundPrecision int
	sharpLimit     float64

	// Texture to use for drawing. If nil, a plain white texture is used.
	Texture *ebiten.Image
}

// NewRibbonTrails creates and initializes a new RibbonTrails object with default values.
func NewRibbonTrails() *RibbonTrails {
	return &RibbonTrails{
		points:            make([]*trailPoint, 0),
		SegmentLifetime:   math.MaxFloat64,
		SegmentWidths:     []float64{10.0},
		SegmentColors:     []color.RGBA{{R: 255, G: 255, B: 255, A: 255}},
		TotalSegmentLimit: 0,
		jointMode:         LineJointBevel, // Bevel is a safe default
		roundPrecision:    8,
		sharpLimit:        2.0,
	}
}

// --- Configuration Setters ---

func (self *RibbonTrails) SetLifetime(lifetime float64) *RibbonTrails {
	if lifetime <= 0 {
		self.SegmentLifetime = math.MaxFloat64
	} else {
		self.SegmentLifetime = lifetime
	}
	return self
}

func (self *RibbonTrails) SetWidths(widths ...float64) *RibbonTrails {
	self.SegmentWidths = widths
	return self
}

func (self *RibbonTrails) SetColors(colors ...color.RGBA) *RibbonTrails {
	self.SegmentColors = colors
	return self
}

func (self *RibbonTrails) SetLimit(limit int) *RibbonTrails {
	self.TotalSegmentLimit = limit
	return self
}

func (self *RibbonTrails) SetJointMode(mode LineJointMode) *RibbonTrails {
	self.jointMode = mode
	return self
}

// --- Core Logic ---

func (self *RibbonTrails) AddPoint(pos ebimath.Vector) {
	// Prevent adding duplicate points, which can cause issues with trail generation.
	if len(self.points) > 0 && self.points[len(self.points)-1].pos.Equals(pos) {
		return
	}
	if self.TotalSegmentLimit > 0 && len(self.points) >= self.TotalSegmentLimit {
		self.points = self.points[1:]
	}
	self.points = append(self.points, &trailPoint{pos: pos, creationTime: self.currentTime})
}

func (self *RibbonTrails) Update(deltaTime float64) {
	self.currentTime += deltaTime

	// 1. Remove old points
	if self.SegmentLifetime != math.MaxFloat64 {
		alivePoints := self.points[:0]
		for _, p := range self.points {
			if self.currentTime-p.creationTime < self.SegmentLifetime {
				alivePoints = append(alivePoints, p)
			}
		}
		self.points = alivePoints
	}

	// 2. Rebuild the mesh from scratch every frame
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]

	if len(self.points) < 2 {
		return
	}

	n := len(self.points)

	for i := 0; i < n-1; i++ {
		pA := self.points[i]
		pB := self.points[i+1]

		// Calculate lifetime-based properties for the segment's start and end points
		progressA := (self.currentTime - pA.creationTime) / self.SegmentLifetime
		progressB := (self.currentTime - pB.creationTime) / self.SegmentLifetime

		colorA := interpolateColor(self.SegmentColors, progressA)
		colorB := interpolateColor(self.SegmentColors, progressB)
		widthA := interpolateWidth(self.SegmentWidths, progressA) / 2.0
		widthB := interpolateWidth(self.SegmentWidths, progressB) / 2.0

		// Build segment
		dirAB := pB.pos.Sub(pA.pos)
		if dirAB.Length() < 1e-6 {
			continue
		}
		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.pos.Add(normal.ScaleF(widthA))
		vA2 := pA.pos.Sub(normal.ScaleF(widthA))
		vB1 := pB.pos.Add(normal.ScaleF(widthB))
		vB2 := pB.pos.Sub(normal.ScaleF(widthB))

		self.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		self.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)

		// Build joint
		if i >= n-2 {
			continue
		}
		pC := self.points[i+2]
		dirBC := pC.pos.Sub(pB.pos)
		if dirBC.Length() < 1e-6 {
			continue
		}

		normalB := dirBC.Normalize().Orthogonal()
		zCross := dirAB.Cross(dirBC)

		if math.Abs(zCross) < 1e-6 {
			continue // Almost straight, no joint needed
		}

		vB1_next := pB.pos.Add(normalB.ScaleF(widthB))
		vB2_next := pB.pos.Sub(normalB.ScaleF(widthB))

		switch self.jointMode {
		case LineJointBevel:
			if zCross < 0 {
				self.addTriangle(pB.pos, vB1, vB1_next, colorB, colorB, colorB)
			} else {
				self.addTriangle(pB.pos, vB2, vB2_next, colorB, colorB, colorB)
			}
		case LineJointRound:
			var vStart, vEnd ebimath.Vector
			if zCross < 0 {
				vStart, vEnd = vB1, vB1_next
			} else {
				vStart, vEnd = vB2, vB2_next
			}
			v1 := vStart.Sub(pB.pos)
			v2 := vEnd.Sub(pB.pos)
			angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
			angleStep := math.Pi / float64(self.roundPrecision)

			if math.Abs(angle) < angleStep {
				// Angle too small for an arc, fall back to bevel.
				if zCross < 0 {
					self.addTriangle(pB.pos, vB1, vB1_next, colorB, colorB, colorB)
				} else {
					self.addTriangle(pB.pos, vB2, vB2_next, colorB, colorB, colorB)
				}
			} else {
				self.addArc(pB.pos, v1, angle, colorB)
			}
		case LineJointSharp:
			var vB_outer, vB_outer_next ebimath.Vector
			if zCross < 0 {
				vB_outer = vB1
				vB_outer_next = vB1_next
			} else {
				vB_outer = vB2
				vB_outer_next = vB2_next
			}
			d := vB_outer_next.Sub(vB_outer)
			u := dirAB
			v := dirBC
			denom := u.Cross(v)
			if math.Abs(denom) < 1e-6 {
				continue
			}
			t := d.Cross(v) / denom
			M := vB_outer.Add(u.ScaleF(t))
			dist := pB.pos.DistanceTo(M)
			if (dist / widthB) > self.sharpLimit {
				self.addTriangle(pB.pos, vB_outer, vB_outer_next, colorB, colorB, colorB)
			} else {
				self.addTriangle(pB.pos, vB_outer, M, colorB, colorB, colorB)
				self.addTriangle(pB.pos, M, vB_outer_next, colorB, colorB, colorB)
			}
		}
	}
}

func (self *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(self.vertices) == 0 {
		return
	}
	texture := self.Texture
	if texture == nil {
		texture = ebiten.NewImage(1, 1)
		texture.Fill(color.White)
	}
	screen.DrawTriangles(self.vertices, self.indices, texture, op)
}

// --- Mesh Generation Helpers (ported from LineBuilder) ---

func (self *RibbonTrails) addTriangle(v1, v2, v3 ebimath.Vector, c1, c2, c3 color.RGBA) {
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

func (self *RibbonTrails) addArc(center, vbegin ebimath.Vector, angleDelta float64, clr color.RGBA) {
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

// interpolateWidth gets the value from a slice based on progress
func interpolateWidth(values []float64, progress float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	if progress <= 0 {
		return values[0]
	}
	if progress >= 1 {
		return values[len(values)-1]
	}

	idx := progress * float64(len(values)-1)
	i1 := int(idx)
	i2 := i1 + 1
	if i2 >= len(values) {
		i2 = len(values) - 1
	}
	t := idx - float64(i1)
	return values[i1]*(1-t) + values[i2]*t
}

// interpolateColor gets the color from a slice based on progress
func interpolateColor(values []color.RGBA, progress float64) color.RGBA {
	if len(values) == 0 {
		return color.RGBA{}
	}
	if len(values) == 1 {
		return values[0]
	}
	if progress <= 0 {
		return values[0]
	}
	if progress >= 1 {
		return values[len(values)-1]
	}

	idx := progress * float64(len(values)-1)
	i1 := int(idx)
	i2 := i1 + 1
	if i2 >= len(values) {
		i2 = len(values) - 1
	}
	t := idx - float64(i1)

	rVal := float64(values[i1].R)*(1-t) + float64(values[i2].R)*t
	gVal := float64(values[i1].G)*(1-t) + float64(values[i2].G)*t
	bVal := float64(values[i1].B)*(1-t) + float64(values[i2].B)*t
	aVal := float64(values[i1].A)*(1-t) + float64(values[i2].A)*t

	return color.RGBA{R: uint8(rVal), G: uint8(gVal), B: uint8(bVal), A: uint8(aVal)}
}
