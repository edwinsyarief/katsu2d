package line

import (
	"image"
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

type LineTextureMode int

const (
	LineTextureNone LineTextureMode = iota
	LineTextureStretch
	LineTextureTile
)

type LineTextureDirection int

const (
	LineTextureHorizontal LineTextureDirection = iota
	LineTextureVertical
)

type LinePoint struct {
	position ebimath.Vector
	color    color.RGBA
	width    float64

	top, bottom ebimath.Vector
}

func createLinePoint(position ebimath.Vector, col color.RGBA, width float64) *LinePoint {
	return &LinePoint{
		position: position,
		color:    col,
		width:    width,
	}
}

type LinePoints []*LinePoint

type Line struct {
	vertices []ebiten.Vertex
	indices  []uint16

	color                color.RGBA
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
	whiteDot             *ebiten.Image

	// isDirty is a flag used for caching. The mesh is only rebuilt when this is true.
	isDirty bool
}

func NewLineBuilder(col color.RGBA, width float64) *Line {
	res := &Line{
		color:       col,
		width:       width,
		points:      LinePoints{},
		tileScale:   1.0,
		textureMode: LineTextureNone,
		vertices:    make([]ebiten.Vertex, 0),
		indices:     make([]uint16, 0),
		// The mesh is initially dirty and needs to be built on the first draw.
		isDirty: true,
	}
	res.whiteDot = ebiten.NewImage(1, 1)
	res.whiteDot.Fill(color.White)

	return res
}

func (self *Line) SetTexture(img *ebiten.Image) {
	if img == nil {
		return
	}
	self.isDirty = true
	self.texture = img
	self.textureSize = ebimath.V(float64(img.Bounds().Dx()), float64(img.Bounds().Dy()))
}

func (self *Line) SetTextureSize(size ebimath.Vector) {
	self.isDirty = true
	self.textureSize = size
}

func (self *Line) SetTextureDirection(direction LineTextureDirection) {
	self.isDirty = true
	self.textureDirection = direction
}

func (self *Line) SetTextureMode(mode LineTextureMode) {
	self.isDirty = true
	self.textureMode = mode
}

func (self *Line) SetTileScale(scale float64) {
	self.isDirty = true
	self.tileScale = scale
}

func (self *Line) SetPointCountLimit(limit int) {
	if limit <= 0 {
		return
	}
	// No need to set dirty if the point limit is the same or the number of points doesn't change.
	// But to be safe and simple, we set it anyway as this changes a property.
	self.isDirty = true
	self.pointLimit = limit
	if len(self.points) > self.pointLimit {
		self.points = self.points[len(self.points)-self.pointLimit:]
	}
}

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

func (self *Line) SetPosition(index int, position ebimath.Vector) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].position = position
}

func (self *Line) SetWidth(index int, width float64) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].width = width
}

func (self *Line) ApplyAllWidth(width float64) {
	self.isDirty = true
	self.interpolateWidth = false
	self.width = width
	for _, s := range self.points {
		s.width = width
	}
}

func (self *Line) InterpolateWidth(start, end float64) {
	self.isDirty = true
	self.interpolateWidth = true
	self.width = start
	self.startWidth = start
	self.endWidth = end
}

func (self *Line) SetColor(index int, color color.RGBA) {
	if index < 0 || index >= len(self.points) {
		return
	}
	self.isDirty = true
	self.points[index].color = color
}

func (self *Line) ApplyAllColor(color color.RGBA) {
	self.isDirty = true
	self.interpolateColor = false
	for i := range self.points {
		self.points[i].color = color
	}
}

func (self *Line) InterpolateColors(colors ...color.RGBA) {
	self.isDirty = true
	self.interpolateColor = true
	self.colors = colors
}

// BuildMesh rebuilds the line mesh only if a change has occurred.
func (self *Line) BuildMesh() {
	// If the line is not dirty, there is no need to rebuild the mesh.
	if !self.isDirty {
		return
	}

	self.ResetMesh()

	if len(self.points) < 2 {
		self.isDirty = false
		return
	}

	currPoint := self.points[len(self.points)-1]
	prevPoint := self.points[len(self.points)-2]

	currHw := currPoint.width / 2
	prevHw := prevPoint.width / 2

	angle := currPoint.position.AngleToPoint(prevPoint.position)
	currPoint.top = currPoint.position.MoveInDirection(currHw, angle+utils.ToRadians(90))
	currPoint.bottom = currPoint.position.MoveInDirection(currHw, angle-utils.ToRadians(90))

	v0 := utils.CreateVertice(ebimath.V2(0), currPoint.top, currPoint.color)
	v1 := utils.CreateVertice(ebimath.V2(0), currPoint.bottom, currPoint.color)

	self.vertices = append(self.vertices, v0, v1)

	angle = prevPoint.position.AngleToPoint(currPoint.position)
	prevPoint.top = prevPoint.position.MoveInDirection(prevHw, angle+utils.ToRadians(90))
	prevPoint.bottom = prevPoint.position.MoveInDirection(prevHw, angle-utils.ToRadians(90))

	v2 := utils.CreateVertice(ebimath.V2(0), prevPoint.top, prevPoint.color)
	v3 := utils.CreateVertice(ebimath.V2(0), prevPoint.bottom, prevPoint.color)

	self.vertices = append(self.vertices, v2, v3)

	for i := len(self.points) - 3; i >= 0; i-- {
		currPoint := self.points[i]
		nextPoint := self.points[i+1]
		furtherPoint := self.points[i+2]

		currHw := currPoint.width / 2
		nextHw := nextPoint.width / 2

		angle := currPoint.position.AngleToPoint(nextPoint.position)
		currPoint.top = currPoint.position.MoveInDirection(currHw, angle+utils.ToRadians(90))
		currPoint.bottom = currPoint.position.MoveInDirection(currHw, angle-utils.ToRadians(90))

		currAngleEnd := nextPoint.position.AngleToPoint(currPoint.position)
		currTopLineStart := currPoint.top
		currBottomLineStart := currPoint.bottom
		currTopLineEnd := nextPoint.position.MoveInDirection(nextHw, currAngleEnd-utils.ToRadians(90))
		currBottomLineEnd := nextPoint.position.MoveInDirection(nextHw, currAngleEnd+utils.ToRadians(90))

		nextTopLineStart := nextPoint.top
		nextBottomLineStart := nextPoint.bottom
		nextTopLineEnd := furtherPoint.top
		nextBottomLineEnd := furtherPoint.bottom

		crossIntersectTop, _ := self.lineIntersects(currTopLineStart, currTopLineEnd, nextBottomLineStart, nextBottomLineEnd)
		crossIntersectBottom, _ := self.lineIntersects(currBottomLineStart, currBottomLineEnd, nextBottomLineStart, nextBottomLineEnd)

		if !crossIntersectTop && !crossIntersectBottom {
			intersectTop, topPos := self.lineIntersects(currTopLineStart, currTopLineEnd, nextTopLineStart, nextTopLineEnd)
			intersectBottom, bottomPos := self.lineIntersects(currBottomLineStart, currBottomLineEnd, nextBottomLineStart, nextBottomLineEnd)
			if intersectTop {
				nextPoint.top = topPos
				dist := topPos.DistanceTo(nextPoint.position)
				if dist > nextPoint.width {
					self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), topPos, currPoint.color))
					self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), currBottomLineEnd, currPoint.color))
				} else {
					angle := topPos.AngleToPoint(nextPoint.position)
					nextPoint.bottom = nextPoint.position.MoveInDirection(dist, angle)
					self.vertices[len(self.vertices)-1].DstX = float32(nextPoint.bottom.X)
					self.vertices[len(self.vertices)-1].DstY = float32(nextPoint.bottom.Y)
					self.vertices[len(self.vertices)-2].DstX = float32(nextPoint.top.X)
					self.vertices[len(self.vertices)-2].DstY = float32(nextPoint.top.Y)
				}
			} else if intersectBottom {
				nextPoint.bottom = bottomPos
				dist := bottomPos.DistanceTo(nextPoint.position)
				if dist > nextPoint.width {
					self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), currTopLineEnd, currPoint.color))
					self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), bottomPos, currPoint.color))
				} else {
					angle := bottomPos.AngleToPoint(nextPoint.position)
					nextPoint.top = nextPoint.position.MoveInDirection(dist, angle)
					self.vertices[len(self.vertices)-1].DstX = float32(nextPoint.bottom.X)
					self.vertices[len(self.vertices)-1].DstY = float32(nextPoint.bottom.Y)
					self.vertices[len(self.vertices)-2].DstX = float32(nextPoint.top.X)
					self.vertices[len(self.vertices)-2].DstY = float32(nextPoint.top.Y)
				}
			}
		}

		self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), currPoint.top, currPoint.color))
		self.vertices = append(self.vertices, utils.CreateVertice(ebimath.V2(0), currPoint.bottom, currPoint.color))
	}

	self.indices = utils.GenerateIndices(len(self.vertices))

	self.calculateUvs()

	// Reset the dirty flag after successfully building the mesh.
	self.isDirty = false
}

func (self *Line) Draw(screen *ebiten.Image) {
	if len(self.vertices) == 0 {
		self.BuildMesh()
	}
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

func (self *Line) ResetMesh() {
	self.vertices = make([]ebiten.Vertex, 0)
	self.indices = make([]uint16, 0)
}

func (self *Line) calculateUvs() {
	if self.textureMode == LineTextureNone {
		return
	}

	segmentCounter := 0
	totalSegments := len(self.colors) - 1
	var segmentSize float64
	if totalSegments > 0 {
		segmentSize = float64(len(self.vertices)/2) / float64(totalSegments)
	}

	for i := 0; i < len(self.vertices); i += 2 {
		if self.interpolateWidth {
			v1 := ebimath.V(float64(self.vertices[i].DstX), float64(self.vertices[i].DstY))
			v2 := ebimath.V(float64(self.vertices[i+1].DstX), float64(self.vertices[i+1].DstY))
			t := float64(segmentCounter) / float64((len(self.vertices)/2)-1)
			width := ebimath.Lerp(self.startWidth, self.endWidth, t) / 2
			center := v1.Add(v2).DivF(2)
			dir := v2.Sub(v1).Normalized()
			dst := center.Sub(dir.MulF(width))
			self.vertices[i].DstX = float32(dst.X)
			self.vertices[i].DstY = float32(dst.Y)
			self.vertices[i+1].DstX = float32(dst.X)
			self.vertices[i+1].DstY = float32(dst.Y)
		}

		if self.interpolateColor && len(self.colors) > 1 {
			segment := int(float64(i) / segmentSize)
			t := (float64(i) / segmentSize) - float64(segment)
			if segment >= totalSegments {
				segment = totalSegments - 1
				t = 1.0
			}
			col := utils.LerpPremultipliedRGBA(self.colors[segment], self.colors[segment+1], t)
			r, g, b, a := col.RGBA()
			self.vertices[i].ColorR = float32(r) / 255
			self.vertices[i].ColorG = float32(g) / 255
			self.vertices[i].ColorB = float32(b) / 255
			self.vertices[i].ColorA = float32(a) / 255
			self.vertices[i+1].ColorR = float32(r) / 255
			self.vertices[i+1].ColorG = float32(g) / 255
			self.vertices[i+1].ColorB = float32(b) / 255
			self.vertices[i+1].ColorA = float32(a) / 255
		}

		switch self.textureMode {
		case LineTextureStretch:
			step := float64(segmentCounter) / float64((len(self.vertices)/2)-1)
			if self.textureDirection == LineTextureHorizontal {
				src1 := ebimath.V(step*self.textureSize.X, 0)
				src2 := ebimath.V(step*self.textureSize.X, self.textureSize.Y)
				self.vertices[i].SrcX = float32(src1.X)
				self.vertices[i].SrcY = float32(src1.Y)
				self.vertices[i+1].SrcX = float32(src2.X)
				self.vertices[i+1].SrcY = float32(src2.Y)
			} else {
				src1 := ebimath.V(0, step*self.textureSize.Y)
				src2 := ebimath.V(self.textureSize.X, step*self.textureSize.Y)
				self.vertices[i].SrcX = float32(src1.X)
				self.vertices[i].SrcY = float32(src1.Y)
				self.vertices[i+1].SrcX = float32(src2.X)
				self.vertices[i+1].SrcY = float32(src2.Y)
			}
		case LineTextureTile:
			if self.textureDirection == LineTextureHorizontal {
				segmentSize := self.textureSize.X / self.tileScale
				step := float64(segmentCounter%2) * segmentSize
				src1 := ebimath.V(step, 0)
				src2 := ebimath.V(step, self.textureSize.Y)
				self.vertices[i].SrcX = float32(src1.X)
				self.vertices[i].SrcY = float32(src1.Y)
				self.vertices[i+1].SrcX = float32(src2.X)
				self.vertices[i+1].SrcY = float32(src2.Y)
			} else {
				segmentSize := self.textureSize.Y / self.tileScale
				step := float64(segmentCounter%2) * segmentSize
				src1 := ebimath.V(0, step)
				src2 := ebimath.V(self.textureSize.X, step)
				self.vertices[i].SrcX = float32(src1.X)
				self.vertices[i].SrcY = float32(src1.Y)
				self.vertices[i+1].SrcX = float32(src2.X)
				self.vertices[i+1].SrcY = float32(src2.Y)
			}
		}

		segmentCounter++
	}
}

func (self *Line) lineIntersects(fromA, toA, fromB, toB ebimath.Vector) (bool, ebimath.Vector) {
	A1 := toA.Y - fromA.Y
	B1 := fromA.X - toA.X
	C1 := A1*fromA.X + B1*fromA.Y

	A2 := toB.Y - fromB.Y
	B2 := fromB.X - toB.X
	C2 := A2*fromB.X + B2*fromB.Y

	det := A1*B2 - A2*B1
	if det != 0 {
		return true, ebimath.V((B2*C1-B1*C2)/det, (A1*C2-A2*C1)/det)
	}
	return false, ebimath.V2(0)
}
