package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
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
// top and bottom are used for building the mesh geometry.
type LinePoint struct {
	position ebimath.Vector
	color    color.RGBA
	width    float64

	top, bottom ebimath.Vector
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
		// The mesh is initially dirty and needs to be built on the first draw.
		isDirty: true,
	}
	res.whiteDot = ebiten.NewImage(1, 1)
	res.whiteDot.Fill(color.White)

	return res
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

	// NEW: Check if width interpolation is enabled and apply it to each point.
	if self.interpolateWidth {
		for i, p := range self.points {
			// Calculate the interpolation factor (t)
			t := float64(i) / float64(len(self.points)-1)

			// Interpolate the width and update the point's width value
			p.width = self.startWidth + t*(self.endWidth-self.startWidth)
		}
	}

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
				// The scaling factor is 1 / sin(angle / 2) which is the miter limit.
				// Since dot product of two normalized vectors is cos(angle), we can get the angle from that.
				cosHalfAngle := math.Sqrt((dir1.Dot(dir2) + 1.0) / 2.0)
				miterScale := p.width / (2.0 * cosHalfAngle)

				miter = miter.Normalized().ScaleF(miterScale)

				p.top = p.position.Add(miter)
				p.bottom = p.position.Sub(miter)
			}
		}
	}

	// Step 2: Build the mesh from the calculated points
	for i := 0; i < len(self.points)-1; i++ {
		p1 := self.points[i]
		p2 := self.points[i+1]

		// Append vertices for the current segment
		v1 := utils.CreateVertexDefaultSrc(p1.top, p1.color)
		v2 := utils.CreateVertexDefaultSrc(p1.bottom, p1.color)
		v3 := utils.CreateVertexDefaultSrc(p2.top, p2.color)
		v4 := utils.CreateVertexDefaultSrc(p2.bottom, p2.color)

		self.vertices = append(self.vertices, v1, v2, v3, v4)

		// Append indices for the quad (2 triangles)
		vertexIndex := uint16(i * 4) // Each segment adds 4 vertices
		self.indices = append(self.indices,
			vertexIndex+0, vertexIndex+1, vertexIndex+2,
			vertexIndex+2, vertexIndex+1, vertexIndex+3,
		)
	}

	self.calculateUvs()
	self.isDirty = false
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

// calculateUvs sets the texture coordinates and interpolates color/width if enabled.
// This function no longer changes DstX/DstY.
func (self *Line) calculateUvs() {
	// The vertices are already correctly positioned by BuildMesh. This function should only handle UVs and colors.
	totalSegments := len(self.points) - 1 // Use points slice for correct interpolation range
	if totalSegments == 0 {
		return
	}

	for i := 0; i < len(self.vertices); i += 4 {
		p1_index := i / 4

		// Handle color interpolation
		if self.interpolateColor {
			t := float64(p1_index) / float64(totalSegments)

			// Interpolate color for the first point of the segment
			segment := int(t * float64(len(self.colors)-1))
			tInSegment := (t * float64(len(self.colors)-1)) - float64(segment)
			if segment >= len(self.colors)-1 {
				segment = len(self.colors) - 2
				tInSegment = 1.0
			}
			col1 := utils.LerpPremultipliedRGBA(self.colors[segment], self.colors[segment+1], tInSegment)

			// Interpolate color for the second point of the segment
			t2 := float64(p1_index+1) / float64(totalSegments)
			segment2 := int(t2 * float64(len(self.colors)-1))
			tInSegment2 := (t2 * float64(len(self.colors)-1)) - float64(segment2)
			if segment2 >= len(self.colors)-1 {
				segment2 = len(self.colors) - 2
				tInSegment2 = 1.0
			}
			col2 := utils.LerpPremultipliedRGBA(self.colors[segment2], self.colors[segment2+1], tInSegment2)

			// Apply interpolated colors to the four vertices of the quad
			self.vertices[i].ColorR, self.vertices[i].ColorG, self.vertices[i].ColorB, self.vertices[i].ColorA = float32(col1.R)/255.0, float32(col1.G)/255.0, float32(col1.B)/255.0, float32(col1.A)/255.0
			self.vertices[i+1].ColorR, self.vertices[i+1].ColorG, self.vertices[i+1].ColorB, self.vertices[i+1].ColorA = float32(col1.R)/255.0, float32(col1.G)/255.0, float32(col1.B)/255.0, float32(col1.A)/255.0
			self.vertices[i+2].ColorR, self.vertices[i+2].ColorG, self.vertices[i+2].ColorB, self.vertices[i+2].ColorA = float32(col2.R)/255.0, float32(col2.G)/255.0, float32(col2.B)/255.0, float32(col2.A)/255.0
			self.vertices[i+3].ColorR, self.vertices[i+3].ColorG, self.vertices[i+3].ColorB, self.vertices[i+3].ColorA = float32(col2.R)/255.0, float32(col2.G)/255.0, float32(col2.B)/255.0, float32(col2.A)/255.0
		}

		// Handle texture coordinates (UVs)
		if self.textureMode != LineTextureNone {
			step := float64(p1_index) / float64(totalSegments)

			// Use a temporary texture size based on mode
			tempTextureSize := ebimath.Vector{}
			if self.textureMode == LineTextureTile {
				tempTextureSize = self.textureSize.ScaleF(self.tileScale)
			} else {
				tempTextureSize = self.textureSize.ScaleF(self.textureScale)
			}

			// Apply UVs based on texture direction
			if self.textureDirection == LineTextureHorizontal {
				self.vertices[i].SrcX, self.vertices[i].SrcY = float32(step*tempTextureSize.X), 0
				self.vertices[i+1].SrcX, self.vertices[i+1].SrcY = float32(step*tempTextureSize.X), float32(tempTextureSize.Y)
				self.vertices[i+2].SrcX, self.vertices[i+2].SrcY = float32((step+1.0/float64(totalSegments))*tempTextureSize.X), 0
				self.vertices[i+3].SrcX, self.vertices[i+3].SrcY = float32((step+1.0/float64(totalSegments))*tempTextureSize.X), float32(tempTextureSize.Y)
			} else { // LineTextureVertical
				self.vertices[i].SrcX, self.vertices[i].SrcY = 0, float32(step*tempTextureSize.Y)
				self.vertices[i+1].SrcX, self.vertices[i+1].SrcY = float32(tempTextureSize.X), float32(step*tempTextureSize.Y)
				self.vertices[i+2].SrcX, self.vertices[i+2].SrcY = 0, float32((step+1.0/float64(totalSegments))*tempTextureSize.Y)
				self.vertices[i+3].SrcX, self.vertices[i+3].SrcY = float32(tempTextureSize.X), float32((step+1.0/float64(totalSegments))*tempTextureSize.Y)
			}
		}

		// Apply opacity to all vertices
		for j := 0; j < 4; j++ {
			self.vertices[i+j].ColorA *= float32(self.opacity)
		}
	}
}
