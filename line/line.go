package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// LineJoinType defines the style of the line joins.
type LineJoinType int

// Constants for different line join styles.
const (
	LineJoinMiter LineJoinType = iota
	LineJoinBevel
	LineJoinRound
)

// LineTextureMode defines how a texture is applied to the line.
type LineTextureMode int

const (
	LineTextureNone LineTextureMode = iota
	LineTextureStretch
	LineTextureTile
)

// LineTextureDirection defines the direction of the texture on the line.
type LineTextureDirection int

const (
	LineTextureHorizontal LineTextureDirection = iota
	LineTextureVertical
)

// LinePoint represents a single point on the line with position, color, and width.
type LinePoint struct {
	position ebimath.Vector
	color    color.RGBA
	width    float64
	// New fields to store the pre-calculated top and bottom vertices for the mesh.
	top    ebimath.Vector
	bottom ebimath.Vector
}

// createLinePoint is a helper to create a new LinePoint.
func createLinePoint(position ebimath.Vector, col color.RGBA, width float64) *LinePoint {
	return &LinePoint{
		position: position,
		color:    col,
		width:    width,
	}
}

// LinePoints is a slice of LinePoint pointers.
type LinePoints []*LinePoint

// Line holds all the data required to render a line as a series of connected quads.
type Line struct {
	vertices []ebiten.Vertex
	indices  []uint16

	color                color.RGBA
	opacity              float64
	points               LinePoints
	width                float64
	textureMode          LineTextureMode
	interpolateColor     bool
	interpolateWidth     bool
	startWidth, endWidth float64
	colors               []color.RGBA
	textureSize          ebimath.Vector
	textureDirection     LineTextureDirection
	tileScale            float64
	pointLimit           int
	texture              *ebiten.Image
	textureScale         float64
	whiteDot             *ebiten.Image
	lineJoin             LineJoinType

	// isDirty is a flag used for caching. The mesh is only rebuilt when this is true.
	isDirty bool
}

// NewLine creates and initializes a new Line object.
func NewLine(col color.RGBA, width float64) *Line {
	res := &Line{
		color:        col,
		opacity:      1.0,
		width:        width,
		points:       LinePoints{},
		tileScale:    1.0,
		textureMode:  LineTextureNone,
		textureScale: 1.0,
		vertices:     make([]ebiten.Vertex, 0),
		indices:      make([]uint16, 0),
		lineJoin:     LineJoinMiter, // Default to Miter join
		// The mesh is initially dirty and needs to be built on the first draw.
		isDirty: true,
	}
	res.whiteDot = ebiten.NewImage(1, 1)
	res.whiteDot.Fill(color.White)

	return res
}

// SetLineJoin sets the style of the line joins.
func (self *Line) SetLineJoin(join LineJoinType) {
	self.isDirty = true
	self.lineJoin = join
}

// SetTexture sets the texture image for the line.
func (self *Line) SetTexture(img *ebiten.Image) {
	if img == nil {
		return
	}
	self.isDirty = true
	self.texture = img
	self.textureSize = ebimath.V(float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
}

// SetTextureSize sets the texture size.
func (self *Line) SetTextureSize(size ebimath.Vector) {
	self.isDirty = true
	self.textureSize = size
}

// SetTextureDirection sets the texture direction.
func (self *Line) SetTextureDirection(direction LineTextureDirection) {
	self.isDirty = true
	self.textureDirection = direction
}

// SetTextureMode sets the texture mode.
func (self *Line) SetTextureMode(mode LineTextureMode) {
	self.isDirty = true
	self.textureMode = mode
}

// SetTileScale sets the tile scale for texture tiling.
func (self *Line) SetTileScale(scale float64) {
	self.isDirty = true
	self.tileScale = scale
}

// SetPointCountLimit sets the maximum number of points the line can hold.
func (self *Line) SetPointCountLimit(limit int) {
	if limit <= 0 {
		return
	}
	self.isDirty = true
	self.pointLimit = limit
	if len(self.points) > self.pointLimit {
		self.points = self.points[len(self.points)-self.pointLimit:]
	}
}

// AddPoint adds a new point to the end of the line.
func (self *Line) AddPoint(pos ebimath.Vector) {
	if len(self.points) > 0 {
		lastPoint := self.points[len(self.points)-1].position
		if lastPoint.Equals(pos) {
			return
		}
	}
	self.isDirty = true
	self.points = append(self.points, createLinePoint(pos, self.color, self.width))

	if self.pointLimit > 0 && len(self.points) > self.pointLimit {
		self.points = self.points[1:]
	}
}

// SetPosition sets the position of a point at a given index.
func (self *Line) SetPosition(index int, position ebimath.Vector) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].position = position
}

// SetWidth sets the width of a point at a given index.
func (self *Line) SetWidth(index int, width float64) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].width = width
}

// ApplyAllWidth sets the width for all points and disables interpolation.
func (self *Line) ApplyAllWidth(width float64) {
	self.isDirty = true
	self.interpolateWidth = false
	self.width = width
	for _, s := range self.points {
		s.width = width
	}
}

// InterpolateWidth enables width interpolation and sets the start and end widths.
func (self *Line) InterpolateWidth(start, end float64) {
	self.isDirty = true
	self.interpolateWidth = true
	self.width = start
	self.startWidth = start
	self.endWidth = end
}

// SetColor sets the color of a point at a given index.
func (self *Line) SetColor(index int, color color.RGBA) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].color = color
}

// ApplyAllColor sets the color for all points and disables interpolation.
func (self *Line) ApplyAllColor(color color.RGBA) {
	self.isDirty = true
	self.interpolateColor = false
	for i := range self.points {
		self.points[i].color = color
	}
}

// InterpolateColors enables color interpolation and sets the colors to use.
func (self *Line) InterpolateColors(colors ...color.RGBA) {
	self.isDirty = true
	self.interpolateColor = true
	self.colors = colors
}

// BuildMesh rebuilds the line mesh only if a change has occurred.
func (self *Line) BuildMesh() {
	if !self.isDirty {
		return
	}
	self.ResetMesh()

	if len(self.points) < 2 {
		self.isDirty = false
		return
	}

	// Calculate interpolated widths if enabled
	if self.interpolateWidth {
		for i, p := range self.points {
			t := float64(i) / float64(len(self.points)-1)
			p.width = self.startWidth + t*(self.endWidth-self.startWidth)
		}
	}

	// Delegate the entire mesh building process to a separate function based on the join type.
	switch self.lineJoin {
	case LineJoinMiter:
		self.buildMiterMesh()
	case LineJoinBevel:
		self.buildBevelMesh()
	case LineJoinRound:
		self.buildRoundMesh()
	}

	self.isDirty = false
}

// buildMiterMesh generates the vertices and indices for a miter-joined line.
// This function now uses a two-pass approach to first calculate geometry and then build the mesh.
func (self *Line) buildMiterMesh() {
	// Step 1: Calculate the top and bottom vectors for each point
	for i := 0; i < len(self.points); i++ {
		p := self.points[i]

		if i == 0 {
			// First point: use the angle to the next point
			nextP := self.points[i+1]
			angle := p.position.AngleToPoint(nextP.position)
			perp := ebimath.V(math.Cos(angle+math.Pi/2), math.Sin(angle+math.Pi/2)).Normalized().ScaleF(p.width / 2)
			p.top = p.position.Add(perp)
			p.bottom = p.position.Sub(perp)
		} else if i == len(self.points)-1 {
			// Last point: use the angle from the previous point
			prevP := self.points[i-1]
			angle := prevP.position.AngleToPoint(p.position)
			perp := ebimath.V(math.Cos(angle+math.Pi/2), math.Sin(angle+math.Pi/2)).Normalized().ScaleF(p.width / 2)
			p.top = p.position.Add(perp)
			p.bottom = p.position.Sub(perp)
		} else {
			// Intermediate points: calculate a proper miter joint
			prevP := self.points[i-1]
			nextP := self.points[i+1]

			// Get direction vectors for the two segments
			dir1 := p.position.Sub(prevP.position).Normalized()
			dir2 := nextP.position.Sub(p.position).Normalized()

			// Get the perpendicular vectors
			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			// Calculate the miter vector. The miter is the sum of the two perpendicular vectors.
			miter := perp1.Add(perp2)
			miterLength := miter.Length()

			// If the miter vector is zero, the lines are parallel. We just use the first perpendicular vector.
			if miterLength == 0 {
				p.top = p.position.Add(perp1.ScaleF(p.width / 2))
				p.bottom = p.position.Sub(perp1.ScaleF(p.width / 2))
			} else {
				// Normalize the miter vector and scale it by the correct length.
				cosHalfAngle := math.Sqrt((dir1.Dot(dir2) + 1.0) / 2.0)
				miterScale := p.width / (2.0 * cosHalfAngle)

				// Implement miter limit to prevent long spikes at sharp angles.
				miterLimit := 10.0 // You can adjust this value
				if miterScale > p.width*miterLimit {
					miterScale = p.width * miterLimit
				}

				miter = miter.Normalized().ScaleF(miterScale)

				p.top = p.position.Add(miter)
				p.bottom = p.position.Sub(miter)
			}
		}
	}

	// Step 2: Build the mesh from the calculated points
	totalSegments := len(self.points) - 1
	for i := 0; i < totalSegments; i++ {
		p1 := self.points[i]
		p2 := self.points[i+1]

		var col1, col2 color.RGBA
		// Handle color interpolation based on the user's logic
		if self.interpolateColor {
			t1 := float64(i) / float64(totalSegments)
			col1 = self.lerpColor(t1)

			t2 := float64(i+1) / float64(totalSegments)
			col2 = self.lerpColor(t2)
		} else {
			col1 = p1.color
			col2 = p2.color
		}

		// Use a temporary texture size based on mode
		tempTextureSize := ebimath.Vector{}
		if self.textureMode == LineTextureTile {
			tempTextureSize = self.textureSize.ScaleF(self.tileScale)
		} else {
			tempTextureSize = self.textureSize.ScaleF(self.textureScale)
		}

		step1 := float64(i) / float64(totalSegments)
		step2 := float64(i+1) / float64(totalSegments)

		// Apply UVs based on texture direction
		var uv1x, uv1y, uv2x, uv2y, uv3x, uv3y, uv4x, uv4y float32
		if self.textureDirection == LineTextureHorizontal {
			uv1x, uv1y = float32(step1*tempTextureSize.X), 0
			uv2x, uv2y = float32(step1*tempTextureSize.X), float32(tempTextureSize.Y)
			uv3x, uv3y = float32(step2*tempTextureSize.X), 0
			uv4x, uv4y = float32(step2*tempTextureSize.X), float32(tempTextureSize.Y)
		} else { // LineTextureVertical
			uv1x, uv1y = 0, float32(step1*tempTextureSize.Y)
			uv2x, uv2y = float32(tempTextureSize.X), float32(step1*tempTextureSize.Y)
			uv3x, uv3y = 0, float32(step2*tempTextureSize.Y)
			uv4x, uv4y = float32(tempTextureSize.X), float32(step2*tempTextureSize.Y)
		}

		// Create vertices using the pre-calculated top/bottom vectors and interpolated colors/UVs.
		v1 := ebiten.Vertex{DstX: float32(p1.top.X), DstY: float32(p1.top.Y), SrcX: uv1x, SrcY: uv1y, ColorR: float32(col1.R) / 255, ColorG: float32(col1.G) / 255, ColorB: float32(col1.B) / 255, ColorA: float32(col1.A) / 255}
		v2 := ebiten.Vertex{DstX: float32(p1.bottom.X), DstY: float32(p1.bottom.Y), SrcX: uv2x, SrcY: uv2y, ColorR: float32(col1.R) / 255, ColorG: float32(col1.G) / 255, ColorB: float32(col1.B) / 255, ColorA: float32(col1.A) / 255}
		v3 := ebiten.Vertex{DstX: float32(p2.top.X), DstY: float32(p2.top.Y), SrcX: uv3x, SrcY: uv3y, ColorR: float32(col2.R) / 255, ColorG: float32(col2.G) / 255, ColorB: float32(col2.B) / 255, ColorA: float32(col2.A) / 255}
		v4 := ebiten.Vertex{DstX: float32(p2.bottom.X), DstY: float32(p2.bottom.Y), SrcX: uv4x, SrcY: uv4y, ColorR: float32(col2.R) / 255, ColorG: float32(col2.G) / 255, ColorB: float32(col2.B) / 255, ColorA: float32(col2.A) / 255}

		// Append vertices for the current segment
		self.vertices = append(self.vertices, v1, v2, v3, v4)

		// Append indices for the quad (2 triangles)
		vertexIndex := uint16(i * 4) // Each segment adds 4 vertices
		self.indices = append(self.indices,
			vertexIndex+0, vertexIndex+1, vertexIndex+2,
			vertexIndex+2, vertexIndex+1, vertexIndex+3,
		)
	}

	// Apply opacity to all vertices after they've been created.
	for i := range self.vertices {
		self.vertices[i].ColorA *= float32(self.opacity)
	}
}

// buildBevelMesh generates the vertices and indices for a bevel-joined line.
// This function contains the full loop to build the mesh for this join type.
func (self *Line) buildBevelMesh() {
	totalSegments := len(self.points) - 1
	for i := 0; i < totalSegments; i++ {
		p1 := self.points[i]
		p2 := self.points[i+1]

		color1, color2 := p1.color, p2.color
		if self.interpolateColor {
			t1 := float64(i) / float64(totalSegments)
			color1 = self.lerpColor(t1)
			t2 := float64(i+1) / float64(totalSegments)
			color2 = self.lerpColor(t2)
		}

		dir := p2.position.Sub(p1.position).Normalized()
		perp := ebimath.V(-dir.Y, dir.X).Normalized()

		tempTextureSize := ebimath.Vector{}
		if self.textureMode == LineTextureTile {
			tempTextureSize = self.textureSize.ScaleF(self.tileScale)
		} else {
			tempTextureSize = self.textureSize.ScaleF(self.textureScale)
		}

		step1 := float64(i) / float64(totalSegments)
		step2 := float64(i+1) / float64(totalSegments)

		uv1x, uv1y := float32(step1*tempTextureSize.X), float32(0)
		uv2x, uv2y := float32(step1*tempTextureSize.X), float32(tempTextureSize.Y)
		uv3x, uv3y := float32(step2*tempTextureSize.X), float32(0)
		uv4x, uv4y := float32(step2*tempTextureSize.X), float32(tempTextureSize.Y)

		if self.textureDirection == LineTextureVertical {
			uv1x, uv1y = float32(0), float32(step1*tempTextureSize.Y)
			uv2x, uv2y = float32(tempTextureSize.X), float32(step1*tempTextureSize.Y)
			uv3x, uv3y = float32(0), float32(step2*tempTextureSize.Y)
			uv4x, uv4y = float32(tempTextureSize.X), float32(step2*tempTextureSize.Y)
		}

		v1 := ebiten.Vertex{DstX: float32(p1.position.X + perp.X*p1.width/2), DstY: float32(p1.position.Y + perp.Y*p1.width/2), SrcX: uv1x, SrcY: uv1y, ColorR: float32(color1.R) / 255, ColorG: float32(color1.G) / 255, ColorB: float32(color1.B) / 255, ColorA: float32(color1.A) / 255 * float32(self.opacity)}
		v2 := ebiten.Vertex{DstX: float32(p1.position.X - perp.X*p1.width/2), DstY: float32(p1.position.Y - perp.Y*p1.width/2), SrcX: uv2x, SrcY: uv2y, ColorR: float32(color1.R) / 255, ColorG: float32(color1.G) / 255, ColorB: float32(color1.B) / 255, ColorA: float32(color1.A) / 255 * float32(self.opacity)}
		v3 := ebiten.Vertex{DstX: float32(p2.position.X + perp.X*p2.width/2), DstY: float32(p2.position.Y + perp.Y*p2.width/2), SrcX: uv3x, SrcY: uv3y, ColorR: float32(color2.R) / 255, ColorG: float32(color2.G) / 255, ColorB: float32(color2.B) / 255, ColorA: float32(color2.A) / 255 * float32(self.opacity)}
		v4 := ebiten.Vertex{DstX: float32(p2.position.X - perp.X*p2.width/2), DstY: float32(p2.position.Y - perp.Y*p2.width/2), SrcX: uv4x, SrcY: uv4y, ColorR: float32(color2.R) / 255, ColorG: float32(color2.G) / 255, ColorB: float32(color2.B) / 255, ColorA: float32(color2.A) / 255 * float32(self.opacity)}

		currentVertexIndex := uint16(len(self.vertices))
		self.vertices = append(self.vertices, v1, v2, v3, v4)
		self.indices = append(self.indices,
			currentVertexIndex+0, currentVertexIndex+1, currentVertexIndex+2,
			currentVertexIndex+2, currentVertexIndex+1, currentVertexIndex+3,
		)

		if i > 0 {
			p0 := self.points[i-1]
			p1 := self.points[i]
			p2 := self.points[i+1]

			color1 := p1.color
			if self.interpolateColor {
				t1 := float64(i) / float64(len(self.points)-1)
				color1 = self.lerpColor(t1)
			}

			dir1 := p1.position.Sub(p0.position).Normalized()
			dir2 := p2.position.Sub(p1.position).Normalized()

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			isConvex := dir1.X*dir2.Y-dir1.Y*dir2.X < 0

			joinVertexIndex := uint16(len(self.vertices))
			self.vertices = append(self.vertices, utils.CreateVertexDefaultSrc(p1.position, color1))

			if isConvex {
				v5 := utils.CreateVertexDefaultSrc(p1.position.Add(perp1.ScaleF(p1.width/2)), color1)
				v6 := utils.CreateVertexDefaultSrc(p1.position.Add(perp2.ScaleF(p1.width/2)), color1)
				self.vertices = append(self.vertices, v5, v6)
				self.indices = append(self.indices, joinVertexIndex, joinVertexIndex+1, joinVertexIndex+2)
			} else {
				v5 := utils.CreateVertexDefaultSrc(p1.position.Sub(perp1.ScaleF(p1.width/2)), color1)
				v6 := utils.CreateVertexDefaultSrc(p1.position.Sub(perp2.ScaleF(p1.width/2)), color1)
				self.vertices = append(self.vertices, v5, v6)
				self.indices = append(self.indices, joinVertexIndex, joinVertexIndex+1, joinVertexIndex+2)
			}
		}
	}
}

// buildRoundMesh generates the vertices and indices for a round-joined line.
// This function contains the full loop to build the mesh for this join type.
func (self *Line) buildRoundMesh() {
	totalSegments := len(self.points) - 1
	for i := 0; i < totalSegments; i++ {
		p1 := self.points[i]
		p2 := self.points[i+1]

		color1, color2 := p1.color, p2.color
		if self.interpolateColor {
			t1 := float64(i) / float64(totalSegments)
			color1 = self.lerpColor(t1)
			t2 := float64(i+1) / float64(totalSegments)
			color2 = self.lerpColor(t2)
		}

		dir := p2.position.Sub(p1.position).Normalized()
		perp := ebimath.V(-dir.Y, dir.X).Normalized()

		tempTextureSize := ebimath.Vector{}
		if self.textureMode == LineTextureTile {
			tempTextureSize = self.textureSize.ScaleF(self.tileScale)
		} else {
			tempTextureSize = self.textureSize.ScaleF(self.textureScale)
		}

		step1 := float64(i) / float64(totalSegments)
		step2 := float64(i+1) / float64(totalSegments)

		uv1x, uv1y := float32(step1*tempTextureSize.X), float32(0)
		uv2x, uv2y := float32(step1*tempTextureSize.X), float32(tempTextureSize.Y)
		uv3x, uv3y := float32(step2*tempTextureSize.X), float32(0)
		uv4x, uv4y := float32(step2*tempTextureSize.X), float32(tempTextureSize.Y)

		if self.textureDirection == LineTextureVertical {
			uv1x, uv1y = float32(0), float32(step1*tempTextureSize.Y)
			uv2x, uv2y = float32(tempTextureSize.X), float32(step1*tempTextureSize.Y)
			uv3x, uv3y = float32(0), float32(step2*tempTextureSize.Y)
			uv4x, uv4y = float32(tempTextureSize.X), float32(step2*tempTextureSize.Y)
		}

		v1 := ebiten.Vertex{DstX: float32(p1.position.X + perp.X*p1.width/2), DstY: float32(p1.position.Y + perp.Y*p1.width/2), SrcX: uv1x, SrcY: uv1y, ColorR: float32(color1.R) / 255, ColorG: float32(color1.G) / 255, ColorB: float32(color1.B) / 255, ColorA: float32(color1.A) / 255 * float32(self.opacity)}
		v2 := ebiten.Vertex{DstX: float32(p1.position.X - perp.X*p1.width/2), DstY: float32(p1.position.Y - perp.Y*p1.width/2), SrcX: uv2x, SrcY: uv2y, ColorR: float32(color1.R) / 255, ColorG: float32(color1.G) / 255, ColorB: float32(color1.B) / 255, ColorA: float32(color1.A) / 255 * float32(self.opacity)}
		v3 := ebiten.Vertex{DstX: float32(p2.position.X + perp.X*p2.width/2), DstY: float32(p2.position.Y + perp.Y*p2.width/2), SrcX: uv3x, SrcY: uv3y, ColorR: float32(color2.R) / 255, ColorG: float32(color2.G) / 255, ColorB: float32(color2.B) / 255, ColorA: float32(color2.A) / 255 * float32(self.opacity)}
		v4 := ebiten.Vertex{DstX: float32(p2.position.X - perp.X*p2.width/2), DstY: float32(p2.position.Y - perp.Y*p2.width/2), SrcX: uv4x, SrcY: uv4y, ColorR: float32(color2.R) / 255, ColorG: float32(color2.G) / 255, ColorB: float32(color2.B) / 255, ColorA: float32(color2.A) / 255 * float32(self.opacity)}

		currentVertexIndex := uint16(len(self.vertices))
		self.vertices = append(self.vertices, v1, v2, v3, v4)
		self.indices = append(self.indices,
			currentVertexIndex+0, currentVertexIndex+1, currentVertexIndex+2,
			currentVertexIndex+2, currentVertexIndex+1, currentVertexIndex+3,
		)

		if i > 0 {
			p0 := self.points[i-1]
			p1 := self.points[i]
			p2 := self.points[i+1]

			color1 := p1.color
			if self.interpolateColor {
				t1 := float64(i) / float64(len(self.points)-1)
				color1 = self.lerpColor(t1)
			}

			dir1 := p1.position.Sub(p0.position).Normalized()
			dir2 := p2.position.Sub(p1.position).Normalized()

			perp1 := ebimath.V(-dir1.Y, dir1.X).Normalized()
			perp2 := ebimath.V(-dir2.Y, dir2.X).Normalized()

			crossProduct := dir1.X*dir2.Y - dir1.Y*dir2.X

			startAngle := math.Atan2(perp1.Y, perp1.X)
			endAngle := math.Atan2(perp2.Y, perp2.X)

			if crossProduct > 0 {
				if endAngle < startAngle {
					endAngle += 2 * math.Pi
				}
			} else {
				if startAngle < endAngle {
					startAngle += 2 * math.Pi
				}
			}

			joinVertexIndex := uint16(len(self.vertices))
			self.vertices = append(self.vertices, utils.CreateVertexDefaultSrc(p1.position, color1))

			const segments = 10
			angleDelta := (endAngle - startAngle) / segments

			for s := 0; s <= segments; s++ {
				arcAngle := startAngle + float64(s)*angleDelta
				arcPerp := ebimath.V(math.Cos(arcAngle), math.Sin(arcAngle)).ScaleF(p1.width / 2)
				arcVertex := utils.CreateVertexDefaultSrc(p1.position.Add(arcPerp), color1)
				self.vertices = append(self.vertices, arcVertex)

				if s > 0 {
					self.indices = append(self.indices,
						joinVertexIndex,
						joinVertexIndex+uint16(s),
						joinVertexIndex+uint16(s)+1,
					)
				}
			}
		}
	}
}

// lerpColor interpolates between two colors.
func (self *Line) lerpColor(t float64) color.RGBA {
	segment := int(t * float64(len(self.colors)-1))
	tInSegment := (t * float64(len(self.colors)-1)) - float64(segment)
	if segment >= len(self.colors)-1 {
		segment = len(self.colors) - 2
		tInSegment = 1.0
	}
	return utils.LerpPremultipliedRGBA(self.colors[segment], self.colors[segment+1], tInSegment)
}

// Draw renders the line on the screen.
func (self *Line) Draw(screen *ebiten.Image) {
	self.BuildMesh()

	if len(self.vertices) == 0 {
		return
	}

	img := self.whiteDot
	if self.textureMode != LineTextureNone && self.texture != nil {
		img = self.texture
	}
	op := &ebiten.DrawTrianglesOptions{}
	screen.DrawTriangles(self.vertices, self.indices, img, op)
}

// GetBounds returns the bounding box of the line.
func (self *Line) GetBounds() image.Rectangle {
	if len(self.points) == 0 {
		return image.Rectangle{}
	}
	minX, minY := math.MaxInt, math.MaxInt
	maxX, maxY := -math.MaxInt, -math.MaxInt
	for _, s := range self.points {
		p := s.position
		pX := int(math.Round(p.X))
		pY := int(math.Round(p.Y))
		if pX < minX {
			minX = pX
		}
		if pY < minY {
			minY = pY
		}
		if pX > maxX {
			maxX = pX
		}
		if pY > maxY {
			maxY = pY
		}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

// ResetMesh clears the vertex and index slices.
func (self *Line) ResetMesh() {
	self.vertices = make([]ebiten.Vertex, 0)
	self.indices = make([]uint16, 0)
}
