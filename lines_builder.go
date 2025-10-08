package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// Rect is a 2D rectangle defined by a position and a size.
type Rect struct {
	Position Vector
	Size     Vector
}

// LineBuilder is responsible for generating the vertices and indices for a line
// based on a series of points and various styling parameters.
type LineBuilder struct {
	// points is a slice of 2D vectors that define the path of the line.
	points []Vector
	// widths is an optional slice to specify varying widths along the line.
	widths []float64
	// colors is an optional slice to specify varying colors along the line.
	colors []color.RGBA
	// vertices stores the generated `ebiten.Vertex` data.
	vertices []ebiten.Vertex
	// indices stores the generated indices for the vertex buffer.
	indices []uint16
	// uvs stores texture coordinates for each vertex.
	uvs []Vector
	// jointMode specifies how line segments are connected (e.g., sharp, bevel, round).
	jointMode LineJointMode
	// beginCapMode specifies the style of the line's starting end.
	beginCapMode LineCapMode
	// endCapMode specifies the style of the line's ending end.
	endCapMode LineCapMode
	// width is the default thickness of the line.
	width float64
	// textureMode specifies how a texture is applied to the line.
	textureMode LineTextureMode
	// sharpLimit determines the threshold for a sharp joint.
	sharpLimit float64
	// roundPrecision defines the number of segments used to approximate round caps and joints.
	roundPrecision int
	// tileAspect is the aspect ratio for texture tiling.
	tileAspect float64
	// defaultColor is the fallback color if no specific colors are provided.
	defaultColor color.RGBA
	// closed indicates if the line forms a closed loop.
	closed bool
	// interpolateColor is a flag to enable color interpolation if multiple colors are provided.
	interpolateColor bool
}

// NewLineBuilder is the constructor for a LineBuilder.
// It initializes a new builder with default values.
func NewLineBuilder() *LineBuilder {
	return &LineBuilder{
		sharpLimit:     2.0,
		roundPrecision: 8,
		tileAspect:     1.0,
		width:          10.0,
		defaultColor:   color.RGBA{R: 102, G: 128, B: 255, A: 255},
	}
}

// Build generates the vertices and indices based on the LineBuilder's properties.
// It handles line segments, caps, and joints.
func (self *LineBuilder) Build() {
	if len(self.points) < 2 {
		return // Not enough points to draw a line.
	}
	// --- Pre-computation for interpolation ---
	// Determine if color or width interpolation is needed and calculate the total line distance.
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
	var currentDistance float64
	// --- Begin Cap ---
	// Generates the geometry for the start of the line if it's not a closed loop.
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
			// Adds a square cap by drawing two triangles extending from the line's end.
			v1_box := v1.Sub(dir.ScaleF(width))
			v2_box := v2.Sub(dir.ScaleF(width))
			self.addTriangle(v1, v2, v1_box, color, color, color)
			self.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			// Adds a round cap by drawing a semicircle using `newArc`.
			self.newArc(pA, v1.Sub(pA), math.Pi, color, Rect{})
		}
	}
	// --- Main loop ---
	// Iterates through each segment of the line to generate geometry.
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
			continue // Skip very short segments.
		}
		tA := 0.0
		tB := 0.0
		// Calculate normalized distance for color and width interpolation.
		if totalDistance > 0 {
			tA = currentDistance / totalDistance
			currentDistance += lenAB
			tB = currentDistance / totalDistance
		}
		// Interpolate color and width for the current segment's endpoints.
		colorA := self.lerpColor(tA)
		colorB := self.lerpColor(tB)
		widthA := halfThickness
		widthB := halfThickness
		if interpolateWidth {
			widthA = self.lerpWidth(tA) * halfThickness
			widthB = self.lerpWidth(tB) * halfThickness
		}
		// Calculate the normal vectors for the line segment's vertices.
		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.Add(normal.ScaleF(widthA))
		vA2 := pA.Sub(normal.ScaleF(widthA))
		vB1 := pB.Add(normal.ScaleF(widthB))
		vB2 := pB.Sub(normal.ScaleF(widthB))
		// Add two triangles to form the rectangular line segment.
		self.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB)
		self.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB)
		// --- Joint ---
		// Handles the geometry for the connection between line segments.
		if !isClosed && i >= n-2 {
			continue // Skip joint calculation for the last segment of an open line.
		}
		pcIndex := (i + 2) % n
		pC := self.points[pcIndex]
		if pC.Equals(pB) {
			continue // Skip if the next point is the same.
		}
		dirBC := pC.Sub(pB)
		if dirBC.Length() < 1e-6 {
			continue // Skip if the next segment is too short.
		}
		normalB := dirBC.Normalize().Orthogonal()
		// The cross product determines the direction of the turn (left or right).
		zCross := dirAB.Cross(dirBC)
		if math.Abs(zCross) > 1e-6 {
			vB1_next := pB.Add(normalB.ScaleF(widthB))
			vB2_next := pB.Sub(normalB.ScaleF(widthB))
			switch self.jointMode {
			case LineJointBevel:
				// Adds a beveled joint with a single triangle to "cut the corner."
				if zCross < 0 {
					self.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
				} else {
					self.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
				}
			case LineJointRound:
				// Adds a round joint by drawing a circular arc.
				var vStart, vEnd Vector
				if zCross < 0 {
					vStart, vEnd = vB1, vB1_next
				} else {
					vStart, vEnd = vB2, vB2_next
				}
				v1 := vStart.Sub(pB)
				v2 := vEnd.Sub(pB)
				angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
				// If the angle is too small for a full arc segment, fall back to a bevel join.
				angleStep := math.Pi / float64(self.roundPrecision)
				if math.Abs(angle) < angleStep {
					if zCross < 0 {
						self.addTriangle(pB, vB1, vB1_next, colorB, colorB, colorB)
					} else {
						self.addTriangle(pB, vB2, vB2_next, colorB, colorB, colorB)
					}
				} else {
					self.newArc(pB, v1, angle, colorB, Rect{})
				}
			case LineJointSharp:
				// Adds a miter joint, falling back to a bevel if the corner is too sharp.
				var vB_outer, vB_outer_next Vector
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
				dist := pB.DistanceTo(M)
				ratio := dist / widthB
				if ratio > self.sharpLimit {
					// Fall back to bevel.
					self.addTriangle(pB, vB_outer, vB_outer_next, colorB, colorB, colorB)
				} else {
					// Add miter triangles.
					self.addTriangle(pB, vB_outer, M, colorB, colorB, colorB)
					self.addTriangle(pB, M, vB_outer_next, colorB, colorB, colorB)
				}
			}
		}
	}
	// --- End Cap ---
	// Generates the geometry for the end of the line if it's not a closed loop.
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
			// Adds a square cap by drawing two triangles extending from the line's end.
			v1_box := v1.Add(dir.ScaleF(width))
			v2_box := v2.Add(dir.ScaleF(width))
			self.addTriangle(v1, v2, v1_box, color, color, color)
			self.addTriangle(v2, v2_box, v1_box, color, color, color)
		case LineCapRound:
			// Adds a round cap by drawing a semicircle.
			self.newArc(pB, v1.Sub(pB), -math.Pi, color, Rect{})
		}
	}
}

// addTriangle appends three vertices and their corresponding indices to the builder's slices.
// The indices are generated sequentially.
func (self *LineBuilder) addTriangle(v1, v2, v3 Vector, c1, c2, c3 color.RGBA) {
	idx := uint16(len(self.vertices))
	// Convert RGBA to normalized float32 for Ebiten vertices.
	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255
	// Append the new vertices.
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3},
	)
	// Append the indices for the new triangle.
	self.indices = append(self.indices, idx, idx+1, idx+2)
}

// lerpWidth performs linear interpolation on the widths slice.
// It returns a interpolated width value based on the normalized distance `t`.
func (self *LineBuilder) lerpWidth(t float64) float64 {
	if len(self.widths) < 2 {
		return 1.0 // Return default scaling if not enough widths provided.
	}
	widths := self.widths
	if self.closed {
		widths = append(widths, self.widths[0])
	}
	// Calculate the index and local interpolation value.
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
	// Perform linear interpolation.
	return widths[idx1] + (widths[idx2]-widths[idx1])*localT
}

// newArc generates the vertices and indices for a circular arc.
// It is used for creating round caps and joints.
func (self *LineBuilder) newArc(center, vbegin Vector, angleDelta float64, clr color.RGBA, uvRect Rect) {
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
	// Add the center vertex for the arc fan.
	self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(center.X), DstY: float32(center.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	if self.textureMode != LineTextureNone {
		self.uvs = append(self.uvs, interpolate(uvRect, V(0.5, 0.5)))
	}
	// Generate vertices along the arc.
	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := center.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(rpos.X), DstY: float32(rpos.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
	}
	// Add indices to form triangles for the arc fan.
	for i := 0; i < steps; i++ {
		self.indices = append(self.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

// lerpColor performs linear interpolation on the colors slice.
// It returns a new color based on the normalized distance `t`.
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
	// Calculate the segment and local interpolation value.
	pos := t * float64(len(colors)-1)
	segment := int(pos)
	tInSegment := pos - float64(segment)
	// Clamp the segment index to the valid range.
	if segment >= len(colors)-1 {
		segment = len(colors) - 2
		tInSegment = 1.0
	}
	if segment < 0 {
		segment = 0
		tInSegment = 0.0
	}
	// Use a utility function to perform the actual color interpolation.
	return LerpPremultipliedRGBA(colors[segment], colors[segment+1], tInSegment)
}

// interpolate performs linear interpolation between the corners of a rectangle.
// This is typically used for texture coordinate generation.
func interpolate(r Rect, v Vector) Vector {
	return V(
		r.Position.X+(r.Position.X+r.Size.X-r.Position.X)*v.X,
		r.Position.Y+(r.Position.Y+r.Size.Y-r.Position.Y)*v.Y,
	)
}
