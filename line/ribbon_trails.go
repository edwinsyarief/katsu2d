package line

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// trailPoint holds a single point's position and creation time in the trail.
type trailPoint struct {
	pos          ebimath.Vector
	creationTime float64
}

// RibbonTrails generates a trail effect using a line that fades and scales over time.
type RibbonTrails struct {
	points            []*trailPoint
	segmentLifetime   float64
	segmentWidths     []float64
	segmentColors     []color.RGBA
	totalSegmentLimit int
	currentTime       float64

	// StepDistance specifies the minimum distance between points after resampling.
	StepDistance float64
	// catmullRom enables Catmull-Rom spline interpolation for smoothing the trail.
	catmullRom      bool
	splinePrecision int

	// debugDraw enables rendering of debug points and lines.
	debugDraw   bool
	debugPoints []*trailPoint

	// vertices and indices store the mesh data for drawing.
	vertices []ebiten.Vertex
	indices  []uint16

	jointMode      LineJointMode
	roundPrecision int
	sharpLimit     float64

	texture  *ebiten.Image
	whiteDot *ebiten.Image
}

// NewRibbonTrails creates and initializes a new RibbonTrails object with default settings.
func NewRibbonTrails() *RibbonTrails {
	// Create a 1x1 white image to use as a default texture.
	whiteDotImg := ebiten.NewImage(1, 1)
	whiteDotImg.Fill(color.White)
	return &RibbonTrails{
		points:            make([]*trailPoint, 0),
		segmentLifetime:   math.MaxFloat64,
		segmentWidths:     []float64{10.0},
		segmentColors:     []color.RGBA{{R: 255, G: 255, B: 255, A: 255}},
		totalSegmentLimit: 0,
		StepDistance:      0.0,
		catmullRom:        false,
		splinePrecision:   10,
		jointMode:         LineJointBevel,
		roundPrecision:    8,
		sharpLimit:        2.0,
		whiteDot:          whiteDotImg,
	}
}

// SetLifetime sets the maximum time a trail segment can exist before it's removed.
func (self *RibbonTrails) SetLifetime(lifetime float64) *RibbonTrails {
	if lifetime <= 0 {
		self.segmentLifetime = math.MaxFloat64 // No lifetime limit
	} else {
		self.segmentLifetime = lifetime
	}
	return self
}

// SetWidths defines the widths to interpolate across the trail's length.
func (self *RibbonTrails) SetWidths(widths ...float64) *RibbonTrails {
	self.segmentWidths = widths
	return self
}

// SetColors defines the colors to interpolate across the trail's length.
func (self *RibbonTrails) SetColors(colors ...color.RGBA) *RibbonTrails {
	self.segmentColors = colors
	return self
}

// SetLimit sets the maximum number of points in the trail.
// A value of 0 means there is no limit.
func (self *RibbonTrails) SetLimit(limit int) *RibbonTrails {
	self.totalSegmentLimit = limit
	return self
}

// SetJointMode sets the style of the line joints (bevel, round, or sharp).
func (self *RibbonTrails) SetJointMode(mode LineJointMode) *RibbonTrails {
	self.jointMode = mode
	return self
}

// SetStepDistance sets the minimum distance between points after resampling.
// This is useful for creating a uniform-looking trail.
func (self *RibbonTrails) SetStepDistance(distance float64) *RibbonTrails {
	self.StepDistance = distance
	return self
}

// SetDebugDraw toggles the drawing of debug points and lines.
func (self *RibbonTrails) SetDebugDraw(enabled bool) *RibbonTrails {
	self.debugDraw = enabled
	return self
}

// EnableCatmullRom enables or disables Catmull-Rom spline interpolation.
// The precision parameter controls the smoothness of the spline.
func (self *RibbonTrails) EnableCatmullRom(enabled bool, precision int) *RibbonTrails {
	self.catmullRom = enabled
	if precision > 0 {
		self.splinePrecision = precision
	}
	return self
}

// AddPoint adds a new point to the trail at the current time.
// It ignores duplicate points and enforces the segment limit.
func (self *RibbonTrails) AddPoint(pos ebimath.Vector) {
	if len(self.points) > 0 && self.points[len(self.points)-1].pos.Equals(pos) {
		return
	}
	// If a limit is set, remove the oldest point before adding a new one.
	if self.totalSegmentLimit > 0 && len(self.points) >= self.totalSegmentLimit {
		self.points = self.points[1:]
	}
	self.points = append(self.points, &trailPoint{pos: pos, creationTime: self.currentTime})
}

// Update progresses the trail's simulation, removing old points and generating the mesh.
func (self *RibbonTrails) Update(deltaTime float64) {
	self.currentTime += deltaTime

	// Remove points that have exceeded their lifetime.
	if self.segmentLifetime != math.MaxFloat64 {
		alivePoints := self.points[:0]
		for _, p := range self.points {
			if self.currentTime-p.creationTime < self.segmentLifetime {
				alivePoints = append(alivePoints, p)
			}
		}
		self.points = alivePoints
	}

	pointsToProcess := self.points
	// Apply Catmull-Rom spline interpolation if enabled and there are enough points.
	if self.catmullRom && len(self.points) >= 4 {
		pointsToProcess = self.generateSplinePoints(self.points, self.splinePrecision)
	}

	self.debugPoints = pointsToProcess
	// Resample points to ensure a minimum step distance.
	if self.StepDistance > 0 && len(pointsToProcess) > 1 {
		self.debugPoints = self.resamplePoints(pointsToProcess, self.StepDistance)
	}

	// Reset mesh data for regeneration.
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]

	if len(self.debugPoints) < 2 {
		return
	}

	n := len(self.debugPoints)
	// Iterate through the points to generate the ribbon mesh.
	for i := 0; i < n-1; i++ {
		pA := self.debugPoints[i]
		pB := self.debugPoints[i+1]

		// Calculate progress for interpolation.
		progressA := (self.currentTime - pA.creationTime) / self.segmentLifetime
		progressB := (self.currentTime - pB.creationTime) / self.segmentLifetime

		// Interpolate colors and widths based on progress.
		colorA := interpolateColor(self.segmentColors, progressA)
		colorB := interpolateColor(self.segmentColors, progressB)
		widthA := interpolateWidth(self.segmentWidths, progressA) / 2.0
		widthB := interpolateWidth(self.segmentWidths, progressB) / 2.0

		dirAB := pB.pos.Sub(pA.pos)
		if dirAB.Length() < 1e-6 {
			continue
		}
		// Calculate the normal vector for line thickness.
		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.pos.Add(normal.ScaleF(widthA))
		vA2 := pA.pos.Sub(normal.ScaleF(widthA))
		vB1 := pB.pos.Add(normal.ScaleF(widthB))
		vB2 := pB.pos.Sub(normal.ScaleF(widthB))

		// Add triangles for the main ribbon segment.
		self.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		self.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)

		// Handle line joints between segments.
		if i >= n-2 {
			continue
		}
		pC := self.debugPoints[i+2]
		dirBC := pC.pos.Sub(pB.pos)
		if dirBC.Length() < 1e-6 {
			continue
		}

		normalB := dirBC.Normalize().Orthogonal()
		zCross := dirAB.Cross(dirBC)
		if math.Abs(zCross) < 1e-6 {
			continue
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

// generateSplinePoints creates new points along a Catmull-Rom spline.
func (self *RibbonTrails) generateSplinePoints(points []*trailPoint, precision int) []*trailPoint {
	if len(points) < 2 {
		return points
	}

	newPoints := make([]*trailPoint, 0)
	// Add the first point directly.
	newPoints = append(newPoints, points[0])

	for i := 0; i < len(points)-1; i++ {
		// Define the 4 control points for the spline segment.
		p0 := points[i]
		if i > 0 {
			p0 = points[i-1]
		}

		p1 := points[i]
		p2 := points[i+1]

		p3 := points[i+1]
		if i < len(points)-2 {
			p3 = points[i+2]
		}

		for j := 1; j <= precision; j++ {
			t := float64(j) / float64(precision)
			// Interpolate position and creation time.
			newPos := catmullRomPoint(p0.pos, p1.pos, p2.pos, p3.pos, t)
			time_t0 := p1.creationTime
			time_t1 := p2.creationTime
			newTime := time_t0 + (time_t1-time_t0)*t

			newPoints = append(newPoints, &trailPoint{pos: newPos, creationTime: newTime})
		}
	}

	return newPoints
}

// resamplePoints ensures a minimum distance between consecutive points.
func (self *RibbonTrails) resamplePoints(points []*trailPoint, step float64) []*trailPoint {
	if len(points) < 2 || step <= 0 {
		return points
	}

	newPoints := make([]*trailPoint, 0, len(points))
	newPoints = append(newPoints, points[0])
	distanceNeeded := step

	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]
		segmentVec := p2.pos.Sub(p1.pos)
		segmentLen := segmentVec.Length()

		if segmentLen < 1e-6 {
			continue
		}

		for segmentLen >= distanceNeeded {
			t := distanceNeeded / segmentLen
			newPos := p1.pos.Lerp(p2.pos, t)
			newTime := p1.creationTime + (p2.creationTime-p1.creationTime)*t
			newPoints = append(newPoints, &trailPoint{pos: newPos, creationTime: newTime})
			segmentLen -= distanceNeeded
			p1 = newPoints[len(newPoints)-1]
			distanceNeeded = step
		}
		distanceNeeded -= segmentLen
	}
	return newPoints
}

// Draw renders the trail's mesh to the screen.
// It also draws debug visualizations if enabled.
func (self *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(self.vertices) > 0 {
		texture := self.texture
		if texture == nil {
			texture = self.whiteDot
		}
		screen.DrawTriangles(self.vertices, self.indices, texture, op)
	}

	// Draw debug points and lines.
	if self.debugDraw && len(self.debugPoints) > 0 {
		debugVerts := make([]ebiten.Vertex, 0)
		debugIndices := make([]uint16, 0)
		red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}

		// Draw small red squares at each debug point.
		for _, p := range self.debugPoints {
			v1 := p.pos.Add(ebimath.V(-2, -2))
			v2 := p.pos.Add(ebimath.V(2, -2))
			v3 := p.pos.Add(ebimath.V(-2, 2))
			v4 := p.pos.Add(ebimath.V(2, 2))
			idx := uint16(len(debugVerts))
			debugVerts = append(debugVerts,
				ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
				ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: float32(red.R) / 255, ColorG: float32(red.G) / 255, ColorB: float32(red.B) / 255, ColorA: float32(red.A) / 255},
			)
			debugIndices = append(debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
		}

		// Draw yellow lines connecting the debug points.
		for i := 0; i < len(self.debugPoints)-1; i++ {
			p1 := self.debugPoints[i].pos
			p2 := self.debugPoints[i+1].pos
			dir := p2.Sub(p1).Normalize()
			normal := dir.Orthogonal().ScaleF(0.5)

			v1 := p1.Add(normal)
			v2 := p1.Sub(normal)
			v3 := p2.Add(normal)
			v4 := p2.Sub(normal)
			idx := uint16(len(debugVerts))
			debugVerts = append(debugVerts,
				ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
				ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: float32(yellow.R) / 255, ColorG: float32(yellow.G) / 255, ColorB: float32(yellow.B) / 255, ColorA: float32(yellow.A) / 255},
			)
			debugIndices = append(debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
		}

		if len(debugVerts) > 0 {
			screen.DrawTriangles(debugVerts, debugIndices, self.whiteDot, op)
		}
	}
}

// addTriangle adds a single triangle to the vertex and index buffers.
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

// addArc creates a series of triangles to form a rounded arc at a joint.
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

// interpolateWidth linearly interpolates between widths based on a progress value (0.0 to 1.0).
func interpolateWidth(values []float64, progress float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
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

// interpolateColor linearly interpolates between colors based on a progress value (0.0 to 1.0).
func interpolateColor(values []color.RGBA, progress float64) color.RGBA {
	if len(values) == 0 {
		return color.RGBA{}
	}
	if len(values) == 1 {
		return values[0]
	}
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
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
