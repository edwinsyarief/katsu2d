package katsu2d

import (
	"image"
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
	points           []Vector
	widths           []float64
	colors           []color.RGBA
	vertices         []ebiten.Vertex
	indices          []uint16
	jointMode        LineJointMode
	beginCapMode     LineCapMode
	endCapMode       LineCapMode
	width            float64
	textureMode      LineTextureMode
	texBounds        image.Rectangle
	sharpLimit       float64
	roundPrecision   int
	tileAspect       float64
	textureScale     float64
	textureOffset    float64
	defaultColor     color.RGBA
	closed           bool
	interpolateColor bool
}

// NewLineBuilder creates a new LineBuilder initialized with default styling values.
func NewLineBuilder() *LineBuilder {
	return &LineBuilder{
		sharpLimit:     2.0,
		roundPrecision: 8,
		tileAspect:     1.0,
		textureScale:   1.0,
		width:          10.0,
		defaultColor:   color.RGBA{R: 102, G: 128, B: 255, A: 255},
	}
}

// Reset clears the builder's geometry buffers and input references while retaining
// slice capacity so the builder can be reused across rebuilds without reallocating.
func (b *LineBuilder) Reset() {
	b.points = nil
	b.widths = nil
	b.colors = nil
	b.vertices = b.vertices[:0]
	b.indices = b.indices[:0]
	b.interpolateColor = false
}

// Build generates the vertices and indices based on the LineBuilder's properties.
// It handles line segments, caps, and joints.
func (b *LineBuilder) Build() {
	b.vertices = b.vertices[:0]
	b.indices = b.indices[:0]

	if len(b.points) < 2 {
		return
	}

	b.interpolateColor = len(b.colors) > 1
	interpolateWidth := len(b.widths) > 1
	textured := b.textureMode != LineTextureNone && !b.texBounds.Empty()

	var texMinX, texW float64
	var vTop, vBot, vCenter float32
	if textured {
		texMinX = float64(b.texBounds.Min.X)
		texW = float64(b.texBounds.Dx())
		texMinY := float64(b.texBounds.Min.Y)
		texH := float64(b.texBounds.Dy())
		if texH <= 1 {
			vCenter = float32(texMinY + texH*0.5)
			vTop = vCenter
			vBot = vCenter
		} else {
			vTop = float32(texMinY + 0.5)
			vBot = float32(float64(b.texBounds.Max.Y) - 0.5)
			vCenter = (vTop + vBot) * 0.5
		}
	}

	texScale := b.textureScale
	if texScale <= 0 {
		texScale = 1.0
	}
	tileAspect := b.tileAspect
	if tileAspect <= 0 {
		tileAspect = 1.0
	}

	var totalDistance float64
	n := len(b.points)
	nSegments := n - 1
	isClosed := b.closed && n > 2
	if isClosed {
		nSegments = n
	}
	if b.interpolateColor || interpolateWidth || textured {
		for i := range nSegments {
			p1 := b.points[i%n]
			p2 := b.points[(i+1)%n]
			totalDistance += p1.DistanceTo(p2)
		}
	}

	halfThickness := b.width / 2.0
	var currentDistance float64
	currentU := texMinX + b.textureOffset*texW
	var uStart float32
	if textured {
		uStart = float32(currentU)
	}

	// Begin Cap for open lines.
	if !isClosed && b.beginCapMode != LineCapNone {
		pA := b.points[0]
		pB := b.points[1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = b.lerpWidth(0) * halfThickness
		}
		clr := b.lerpColor(0)
		v1 := pA.Add(normal.ScaleF(width))
		v2 := pA.Sub(normal.ScaleF(width))
		switch b.beginCapMode {
		case LineCapBox:
			v1Box := v1.Sub(dir.ScaleF(width))
			v2Box := v2.Sub(dir.ScaleF(width))
			b.addTriangle(v1, v2, v1Box, clr, clr, clr, uStart, vTop, uStart, vBot, uStart, vTop)
			b.addTriangle(v2, v2Box, v1Box, clr, clr, clr, uStart, vBot, uStart, vBot, uStart, vTop)
		case LineCapRound:
			b.newArc(pA, v1.Sub(pA), math.Pi, clr, uStart, vCenter, vTop, vBot)
		}
	}

	roundPrec := b.roundPrecision
	if roundPrec <= 0 {
		roundPrec = 8
	}
	angleStep := math.Pi / float64(roundPrec)

	// Main segment and joint loop.
	for i := range nSegments {
		paIndex := i % n
		pbIndex := (i + 1) % n
		pA := b.points[paIndex]
		pB := b.points[pbIndex]
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

		colorA := b.lerpColor(tA)
		colorB := b.lerpColor(tB)
		widthA := halfThickness
		widthB := halfThickness
		if interpolateWidth {
			widthA = b.lerpWidth(tA) * halfThickness
			widthB = b.lerpWidth(tB) * halfThickness
		}

		var uA, uB float32
		if textured {
			if b.textureMode == LineTextureStretch {
				uA = float32(texMinX + (tA+b.textureOffset)*texW)
				uB = float32(texMinX + (tB+b.textureOffset)*texW)
			} else {
				segWidth := widthA + widthB
				du := (lenAB / math.Max(1e-4, segWidth*tileAspect)) * texW * texScale
				uA = float32(currentU)
				currentU += du
				uB = float32(currentU)
			}
		}

		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.Add(normal.ScaleF(widthA))
		vA2 := pA.Sub(normal.ScaleF(widthA))
		vB1 := pB.Add(normal.ScaleF(widthB))
		vB2 := pB.Sub(normal.ScaleF(widthB))
		b.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB, uA, vTop, uA, vBot, uB, vTop)
		b.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB, uA, vBot, uB, vBot, uB, vTop)

		if !isClosed && i >= n-2 {
			continue
		}

		pcIndex := (i + 2) % n
		pC := b.points[pcIndex]
		if pC.Equals(pB) {
			continue
		}
		dirBC := pC.Sub(pB)
		if dirBC.Length() < 1e-6 {
			continue
		}

		normalB := dirBC.Normalize().Orthogonal()
		zCross := dirAB.Cross(dirBC)
		if math.Abs(zCross) <= 1e-6 {
			continue
		}

		vB1Next := pB.Add(normalB.ScaleF(widthB))
		vB2Next := pB.Sub(normalB.ScaleF(widthB))
		switch b.jointMode {
		case LineJointBevel:
			if zCross < 0 {
				b.addTriangle(pB, vB1, vB1Next, colorB, colorB, colorB, uB, vCenter, uB, vTop, uB, vTop)
			} else {
				b.addTriangle(pB, vB2, vB2Next, colorB, colorB, colorB, uB, vCenter, uB, vBot, uB, vBot)
			}
		case LineJointRound:
			vStart, vEnd := vB2, vB2Next
			vEdge := vBot
			if zCross < 0 {
				vStart, vEnd = vB1, vB1Next
				vEdge = vTop
			}
			v1 := vStart.Sub(pB)
			v2 := vEnd.Sub(pB)
			angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
			if math.Abs(angle) < angleStep {
				b.addTriangle(pB, vStart, vEnd, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			} else {
				b.newArc(pB, v1, angle, colorB, uB, vCenter, vEdge, vEdge)
			}
		case LineJointSharp:
			vBOuter, vBOuterNext := vB2, vB2Next
			vEdge := vBot
			if zCross < 0 {
				vBOuter, vBOuterNext = vB1, vB1Next
				vEdge = vTop
			}
			d := vBOuterNext.Sub(vBOuter)
			denom := dirAB.Cross(dirBC)
			if math.Abs(denom) < 1e-6 {
				continue
			}
			t := d.Cross(dirBC) / denom
			miter := vBOuter.Add(dirAB.ScaleF(t))
			dist := pB.DistanceTo(miter)
			if widthB <= 0 || dist/widthB > b.sharpLimit {
				b.addTriangle(pB, vBOuter, vBOuterNext, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			} else {
				b.addTriangle(pB, vBOuter, miter, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
				b.addTriangle(pB, miter, vBOuterNext, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			}
		}
	}

	// End Cap for open lines.
	if !isClosed && b.endCapMode != LineCapNone {
		pA := b.points[len(b.points)-2]
		pB := b.points[len(b.points)-1]
		dir := pB.Sub(pA).Normalize()
		normal := dir.Orthogonal()
		width := halfThickness
		if interpolateWidth {
			width = b.lerpWidth(1.0) * halfThickness
		}
		clr := b.lerpColor(1.0)
		var uEnd float32
		if textured {
			if b.textureMode == LineTextureStretch {
				if totalDistance > 0 {
					uEnd = float32(texMinX + (1.0+b.textureOffset)*texW)
				} else {
					uEnd = uStart
				}
			} else {
				uEnd = float32(currentU)
			}
		}
		v1 := pB.Add(normal.ScaleF(width))
		v2 := pB.Sub(normal.ScaleF(width))
		switch b.endCapMode {
		case LineCapBox:
			v1Box := v1.Add(dir.ScaleF(width))
			v2Box := v2.Add(dir.ScaleF(width))
			b.addTriangle(v1, v2, v1Box, clr, clr, clr, uEnd, vTop, uEnd, vBot, uEnd, vTop)
			b.addTriangle(v2, v2Box, v1Box, clr, clr, clr, uEnd, vBot, uEnd, vBot, uEnd, vTop)
		case LineCapRound:
			b.newArc(pB, v1.Sub(pB), -math.Pi, clr, uEnd, vCenter, vTop, vBot)
		}
	}
}

func (b *LineBuilder) addTriangle(v1, v2, v3 Vector, c1, c2, c3 color.RGBA, u1, sv1, u2, sv2, u3, sv3 float32) {
	const inv255 = float32(1.0 / 255.0)
	idx := uint16(len(b.vertices))
	b.vertices = append(b.vertices,
		ebiten.Vertex{
			DstX: float32(v1.X), DstY: float32(v1.Y),
			SrcX: u1, SrcY: sv1,
			ColorR: float32(c1.R) * inv255, ColorG: float32(c1.G) * inv255, ColorB: float32(c1.B) * inv255, ColorA: float32(c1.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v2.X), DstY: float32(v2.Y),
			SrcX: u2, SrcY: sv2,
			ColorR: float32(c2.R) * inv255, ColorG: float32(c2.G) * inv255, ColorB: float32(c2.B) * inv255, ColorA: float32(c2.A) * inv255,
		},
		ebiten.Vertex{
			DstX: float32(v3.X), DstY: float32(v3.Y),
			SrcX: u3, SrcY: sv3,
			ColorR: float32(c3.R) * inv255, ColorG: float32(c3.G) * inv255, ColorB: float32(c3.B) * inv255, ColorA: float32(c3.A) * inv255,
		},
	)
	b.indices = append(b.indices, idx, idx+1, idx+2)
}

func (b *LineBuilder) lerpWidth(t float64) float64 {
	n := len(b.widths)
	if n < 2 {
		return 1.0
	}
	segments := n - 1
	if b.closed {
		segments = n
	}
	pos := t * float64(segments)
	idx1 := int(pos)
	if idx1 < 0 {
		return b.widths[0]
	}
	if idx1 >= segments {
		if b.closed {
			return b.widths[0]
		}
		return b.widths[n-1]
	}
	idx2 := (idx1 + 1) % n
	localT := pos - float64(idx1)
	return b.widths[idx1] + (b.widths[idx2]-b.widths[idx1])*localT
}

func (b *LineBuilder) newArc(center, vbegin Vector, angleDelta float64, clr color.RGBA, u, vCenter, vStart, vEnd float32) {
	const inv255 = float32(1.0 / 255.0)
	prec := b.roundPrecision
	if prec <= 0 {
		prec = 8
	}
	radius := vbegin.Length()
	angleStep := math.Pi / float64(prec)
	steps := int(math.Abs(angleDelta) / angleStep)
	if angleDelta < 0 {
		angleStep = -angleStep
	}
	t := vbegin.Angle()
	cr := float32(clr.R) * inv255
	cg := float32(clr.G) * inv255
	cb := float32(clr.B) * inv255
	ca := float32(clr.A) * inv255

	invSteps := float32(0)
	if steps > 0 {
		invSteps = 1.0 / float32(steps)
	}

	vi := len(b.vertices)
	b.vertices = append(b.vertices, ebiten.Vertex{
		DstX: float32(center.X), DstY: float32(center.Y),
		SrcX: u, SrcY: vCenter,
		ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
	})
	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		frac := float32(i) * invSteps
		if i == steps {
			angle = t + angleDelta
			frac = 1
		}
		v := vStart + (vEnd-vStart)*frac
		rpos := center.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		b.vertices = append(b.vertices, ebiten.Vertex{
			DstX: float32(rpos.X), DstY: float32(rpos.Y),
			SrcX: u, SrcY: v,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})
	}
	for i := range steps {
		b.indices = append(b.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func (b *LineBuilder) lerpColor(t float64) color.RGBA {
	n := len(b.colors)
	if !b.interpolateColor || n == 0 {
		if n == 1 {
			return b.colors[0]
		}
		return b.defaultColor
	}
	if n == 1 {
		return b.colors[0]
	}
	segments := n - 1
	if b.closed {
		segments = n
	}
	pos := t * float64(segments)
	segment := int(pos)
	tInSegment := pos - float64(segment)
	if segment >= segments {
		segment = segments - 1
		tInSegment = 1.0
	}
	if segment < 0 {
		segment = 0
		tInSegment = 0.0
	}
	nextSegment := (segment + 1) % n
	return LerpPremultipliedRGBA(b.colors[segment], b.colors[nextSegment], tInSegment)
}
