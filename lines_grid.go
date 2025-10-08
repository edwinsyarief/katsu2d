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
	// whiteDot is a single-pixel image used as a texture for drawing the colored vertices.
	whiteDot *ebiten.Image
	// vertices holds the slice of ebiten.Vertex objects that make up the grid's mesh.
	vertices []ebiten.Vertex
	// indices holds the slice of uint16 indices that define the triangles from the vertices slice.
	indices []uint16
	// position is the top-left coordinate of the grid in the world.
	position Vector
	// scale is the scaling vector applied to the grid along the X and Y axes.
	scale Vector
	// size is the pixel size of a single cell in the grid (e.g., a 10x10 grid with a size of 10 would be 100x100 pixels).
	size int
	// rows is the number of horizontal grid lines.
	rows int
	// cols is the number of vertical grid lines.
	cols int
	// thickness is the width of the grid lines in pixels.
	thickness float64
	// rotation is the angle of rotation for the entire grid in radians.
	rotation float64
	// cornerTL specifies the join type for the top-left corner.
	cornerTL GridCornerJoinType
	// cornerTR specifies the join type for the top-right corner.
	cornerTR GridCornerJoinType
	// cornerBL specifies the join type for the bottom-left corner.
	cornerBL GridCornerJoinType
	// cornerBR specifies the join type for the bottom-right corner.
	cornerBR GridCornerJoinType
	// color is the single color applied to the entire grid if color interpolation is not used.
	color color.RGBA
	// topLeftColor is the color for the top-left corner for color interpolation.
	topLeftColor color.RGBA
	// topRightColor is the color for the top-right corner for color interpolation.
	topRightColor color.RGBA
	// bottomLeftColor is the color for the bottom-left corner for color interpolation.
	bottomLeftColor color.RGBA
	// bottomRightColor is the color for the bottom-right corner for color interpolation.
	bottomRightColor color.RGBA
	// isColorInterpolated determines if color interpolation is enabled.
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
		vertices:  make([]ebiten.Vertex, 0),
		indices:   make([]uint16, 0),
		whiteDot:  ebiten.NewImage(1, 1),
		rotation:  0,
		scale:     V(1, 1),
		cornerTL:  GridCornerSharp,
		cornerTR:  GridCornerSharp,
		cornerBL:  GridCornerSharp,
		cornerBR:  GridCornerSharp,
	}
	// The single-pixel image is filled with white, and the vertex colors will tint it.
	g.whiteDot.Fill(color.White)
	// Build the initial mesh from the given properties.
	g.buildMesh()
	return g
}

// buildMesh regenerates the entire vertex and index mesh for the grid.
// This function is called whenever a property that affects the grid's geometry (like position, size, or color) is changed.
func (self *GridLine) buildMesh() {
	// Clear existing mesh data to rebuild from scratch.
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	// Calculate the total dimensions of the grid.
	_ = self.thickness / 2.0
	totalWidth := float64(self.cols * self.size)
	totalHeight := float64(self.rows * self.size)
	center := V(totalWidth/2, totalHeight/2)
	// Inner Horizontal lines
	// Add line segments for all internal horizontal grid lines.
	for i := 1; i < self.rows; i++ {
		p1 := Vector{X: 0, Y: float64(i * self.size)}
		p2 := Vector{X: totalWidth, Y: float64(i * self.size)}
		self.addLine(p1, p2, center)
	}
	// Inner Vertical lines
	// Add line segments for all internal vertical grid lines.
	for i := 1; i < self.cols; i++ {
		p1 := Vector{X: float64(i * self.size), Y: 0}
		p2 := Vector{X: float64(i * self.size), Y: totalHeight}
		self.addLine(p1, p2, center)
	}
	// Outer lines
	// Add line segments for the four outer boundary lines of the grid.
	p1 := V(0, 0)
	p2 := V(totalWidth, 0)
	self.addLine(p1, p2, center)
	p1 = V(0, totalHeight)
	p2 = V(totalWidth, totalHeight)
	self.addLine(p1, p2, center)
	p1 = V(0, 0)
	p2 = V(0, totalHeight)
	self.addLine(p1, p2, center)
	p1 = V(totalWidth, 0)
	p2 = V(totalWidth, totalHeight)
	self.addLine(p1, p2, center)
	// Add joints for corners
	// Add the appropriate corner geometry based on the specified corner join types.
	corner := V(0, 0)
	clr := self.getColor(corner)
	dirIn := V(0, -1)
	dirOut := V(1, 0)
	self.addJoint(corner, dirIn, dirOut, self.cornerTL, clr, center)
	corner = V(totalWidth, 0)
	clr = self.getColor(corner)
	dirIn = V(1, 0)
	dirOut = V(0, 1)
	self.addJoint(corner, dirIn, dirOut, self.cornerTR, clr, center)
	corner = V(totalWidth, totalHeight)
	clr = self.getColor(corner)
	dirIn = V(0, 1)
	dirOut = V(-1, 0)
	self.addJoint(corner, dirIn, dirOut, self.cornerBR, clr, center)
	corner = V(0, totalHeight)
	clr = self.getColor(corner)
	dirIn = V(-1, 0)
	dirOut = V(0, -1)
	self.addJoint(corner, dirIn, dirOut, self.cornerBL, clr, center)
}

// addJoint creates the triangles needed to form the specified corner joint type.
// It handles sharp, bevel, and round joints.
func (self *GridLine) addJoint(corner, dirIn, dirOut Vector, joinType GridCornerJoinType, clr color.RGBA, cntr Vector) {
	dirAB := dirIn.Normalize()
	dirBC := dirOut.Normalize()
	normal := dirAB.Orthogonal()
	normalB := dirBC.Orthogonal()
	widthB := self.thickness / 2.0
	vB1 := corner.Add(normal.ScaleF(widthB))
	vB2 := corner.Sub(normal.ScaleF(widthB))
	vB1_next := corner.Add(normalB.ScaleF(widthB))
	vB2_next := corner.Sub(normalB.ScaleF(widthB))
	zCross := dirAB.Cross(dirBC)
	var vStart, vEnd Vector
	if zCross < 0 {
		vStart, vEnd = vB1, vB1_next
	} else {
		vStart, vEnd = vB2, vB2_next
	}
	switch joinType {
	case GridCornerSharp:
		var vB_outer, vB_outer_next Vector
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
		self.addTriangleLocal(corner, vB_outer, M, clr, clr, clr, cntr)
		self.addTriangleLocal(corner, M, vB_outer_next, clr, clr, clr, cntr)
	case GridCornerBevel:
		self.addTriangleLocal(corner, vStart, vEnd, clr, clr, clr, cntr)
	case GridCornerRound:
		v1 := vStart.Sub(corner)
		v2 := vEnd.Sub(corner)
		angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
		self.newArc(corner, v1, angle, clr, cntr)
	}
}

// addTriangleLocal adds a single triangle to the grid's mesh, applying local transformations and color data.
func (self *GridLine) addTriangleLocal(v1, v2, v3 Vector, c1, c2, c3 color.RGBA, center Vector) {
	// Apply transformations (scale, rotation, position) to each vertex.
	v1t := self.transformPos(v1, center)
	v2t := self.transformPos(v2, center)
	v3t := self.transformPos(v3, center)
	// Convert RGBA colors to a float32 representation (0.0 - 1.0) for the vertices.
	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255
	// Get the current index for the first vertex to define the new triangle.
	idx := uint16(len(self.vertices))
	// Append the three new vertices to the main vertices slice.
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: float32(v1t.X), DstY: float32(v1t.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v2t.X), DstY: float32(v2t.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v3t.X), DstY: float32(v3t.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3, SrcX: 0, SrcY: 0},
	)
	// Append the indices to define the new triangle from the appended vertices.
	self.indices = append(self.indices, idx, idx+1, idx+2)
}

// newArc generates the triangles for a rounded corner. It uses a series of small triangles to approximate a smooth arc.
func (self *GridLine) newArc(centerPos, vbegin Vector, angleDelta float64, clr color.RGBA, center Vector) {
	radius := vbegin.Length()
	// Define the step size for the arc's segments.
	angleStep := math.Pi / 8.0
	// Calculate the number of steps needed to draw the arc.
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}
	t := vbegin.Angle()
	r, gcol, b, a := clr.R, clr.G, clr.B, clr.A
	cr, cg, cb, ca := float32(r)/255, float32(gcol)/255, float32(b)/255, float32(a)/255
	// Store the current vertex count to use as a starting index for the new triangles.
	vi := len(self.vertices)
	// Add the center of the arc as the first vertex.
	centert := self.transformPos(centerPos, center)
	self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(centert.X), DstY: float32(centert.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0})
	// Generate vertices along the arc's curve.
	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		// Ensure the last angle is exactly angleDelta to prevent minor inaccuracies.
		if i == steps {
			angle = t + angleDelta
		}
		rpos := centerPos.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		rpost := self.transformPos(rpos, center)
		self.vertices = append(self.vertices, ebiten.Vertex{DstX: float32(rpost.X), DstY: float32(rpost.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0})
	}
	// Create triangles by connecting the center vertex to pairs of arc vertices.
	for i := 0; i < steps; i++ {
		self.indices = append(self.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

// transformPos applies the grid's scale, rotation, and position to a given local vector.
func (self *GridLine) transformPos(pos, center Vector) Vector {
	// Transform in the following order: scale, rotate, then translate.
	scaledCenter := center.Scale(self.scale)
	return pos.Scale(self.scale).Sub(scaledCenter).Rotate(self.rotation).Add(scaledCenter).Add(self.position)
}

// addLine generates and adds the vertices and indices for a single thick line segment to the grid's mesh.
// A line is represented as a rectangle made of two triangles.
func (self *GridLine) addLine(p1, p2, center Vector) {
	halfThick := self.thickness / 2.0
	dir := p2.Sub(p1)
	if dir.Length() == 0 {
		return
	}
	dir = dir.Normalize()
	normal := dir.Orthogonal()
	// Calculate the four corners of the line rectangle.
	lv1 := p1.Add(normal.ScaleF(halfThick))
	lv2 := p1.Sub(normal.ScaleF(halfThick))
	lv3 := p2.Add(normal.ScaleF(halfThick))
	lv4 := p2.Sub(normal.ScaleF(halfThick))
	// Transform the local coordinates to world coordinates.
	v1 := self.transformPos(lv1, center)
	v2 := self.transformPos(lv2, center)
	v3 := self.transformPos(lv3, center)
	v4 := self.transformPos(lv4, center)
	// Determine the color for each vertex, either a single color or an interpolated one.
	var c1, c2, c3, c4 color.RGBA
	if self.isColorInterpolated {
		totalWidth := float64(self.cols * self.size)
		totalHeight := float64(self.rows * self.size)
		c1 = self.calculateInterpolatedColor(lv1, totalWidth, totalHeight)
		c2 = self.calculateInterpolatedColor(lv2, totalWidth, totalHeight)
		c3 = self.calculateInterpolatedColor(lv3, totalWidth, totalHeight)
		c4 = self.calculateInterpolatedColor(lv4, totalWidth, totalHeight)
	} else {
		c1, c2, c3, c4 = self.color, self.color, self.color, self.color
	}
	// Convert colors to float32.
	r1, g1, b1, a1 := c1.R, c1.G, c1.B, c1.A
	cr1, cg1, cb1, ca1 := float32(r1)/255, float32(g1)/255, float32(b1)/255, float32(a1)/255
	r2, g2, b2, a2 := c2.R, c2.G, c2.B, c2.A
	cr2, cg2, cb2, ca2 := float32(r2)/255, float32(g2)/255, float32(b2)/255, float32(a2)/255
	r3, g3, b3, a3 := c3.R, c3.G, c3.B, c3.A
	cr3, cg3, cb3, ca3 := float32(r3)/255, float32(g3)/255, float32(b3)/255, float32(a3)/255
	r4, g4, b4, a4 := c4.R, c4.G, c4.B, c4.A
	cr4, cg4, cb4, ca4 := float32(r4)/255, float32(g4)/255, float32(b4)/255, float32(a4)/255
	// Get the starting index for the new vertices.
	idx := uint16(len(self.vertices))
	// Append the four vertices for the line rectangle.
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr1, ColorG: cg1, ColorB: cb1, ColorA: ca1, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr2, ColorG: cg2, ColorB: cb2, ColorA: ca2, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr3, ColorG: cg3, ColorB: cb3, ColorA: ca3, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: cr4, ColorG: cg4, ColorB: cb4, ColorA: ca4, SrcX: 0, SrcY: 0},
	)
	// Append the six indices to form two triangles from the four vertices.
	self.indices = append(self.indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

// getColor returns the color for a given position on the grid.
// If color interpolation is enabled, it calculates the interpolated color; otherwise, it returns the single grid color.
func (self *GridLine) getColor(pos Vector) color.RGBA {
	if !self.isColorInterpolated {
		return self.color
	}
	totalWidth := float64(self.cols * self.size)
	totalHeight := float64(self.rows * self.size)
	return self.calculateInterpolatedColor(pos, totalWidth, totalHeight)
}

// calculateInterpolatedColor performs bilinear interpolation to determine the color at a given position within the grid.
// The interpolation is based on the four corner colors.
func (self *GridLine) calculateInterpolatedColor(pos Vector, totalWidth, totalHeight float64) color.RGBA {
	// Calculate the normalized position (0.0 to 1.0) of the point within the grid.
	tx := pos.X / totalWidth
	ty := pos.Y / totalHeight
	// Clamp the normalized coordinates to ensure they are within the 0.0 to 1.0 range.
	tx = math.Max(0, math.Min(1, tx))
	ty = math.Max(0, math.Min(1, ty))
	// Perform linear interpolation horizontally (top colors and bottom colors).
	topColor := LerpPremultipliedRGBA(self.topLeftColor, self.topRightColor, tx)
	bottomColor := LerpPremultipliedRGBA(self.bottomLeftColor, self.bottomRightColor, tx)
	// Perform linear interpolation vertically on the results to get the final color.
	finalColor := LerpPremultipliedRGBA(topColor, bottomColor, ty)
	return finalColor
}

// SetColor sets the single, uniform color for the grid.
// It also triggers a rebuild of the mesh to apply the new color.
func (self *GridLine) SetColor(c color.RGBA) {
	self.color = c
	self.buildMesh()
}

// SetPosition sets the top-left position of the grid.
// This triggers a rebuild of the mesh to apply the new position.
func (self *GridLine) SetPosition(pos Vector) {
	self.position = pos
	self.buildMesh()
}

// SetRotation sets the rotation angle for the grid in radians.
// This triggers a rebuild of the mesh to apply the new rotation.
func (self *GridLine) SetRotation(angle float64) {
	self.rotation = angle
	self.buildMesh()
}

// SetScale sets the scaling factor for the grid along the X and Y axes.
// This triggers a rebuild of the mesh to apply the new scale.
func (self *GridLine) SetScale(scale Vector) {
	self.scale = scale
	self.buildMesh()
}

// InterpolateColor sets the corner colors for gradient interpolation across the grid.
// It also enables color interpolation and triggers a mesh rebuild.
func (self *GridLine) InterpolateColor(topLeft, topRight, bottomLeft, bottomRight color.RGBA) {
	self.isColorInterpolated = true
	self.topLeftColor = topLeft
	self.topRightColor = topRight
	self.bottomLeftColor = bottomLeft
	self.bottomRightColor = bottomRight
	self.buildMesh()
}

// SetCornerJoin sets the join types for the four corners of the grid.
// This triggers a mesh rebuild to apply the new corner styles.
func (self *GridLine) SetCornerJoin(tl, tr, bl, br GridCornerJoinType) {
	self.cornerTL = tl
	self.cornerTR = tr
	self.cornerBL = bl
	self.cornerBR = br
	self.buildMesh()
}

// Draw renders the grid's mesh to the screen using a single DrawTriangles call.
// This is an efficient way to draw a large number of lines.
func (self *GridLine) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	// Avoid drawing if there are no vertices in the mesh.
	if len(self.vertices) == 0 {
		return
	}
	// Use ebiten's optimized triangle drawing function.
	screen.DrawTriangles(self.vertices, self.indices, self.whiteDot, op)
}
