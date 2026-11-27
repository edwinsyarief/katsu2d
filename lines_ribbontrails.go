package katsu2d

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// trailPoint holds a single point's position and creation time in the trail.
type trailPoint struct {
	pos          Vector
	creationTime float64
}

// RibbonTrails generates a trail effect using a line that fades and scales over time.
type RibbonTrails struct {
	texture       *ebiten.Image
	whiteDot      *ebiten.Image
	points        []trailPoint
	segmentWidths []float64
	segmentColors []color.RGBA
	debugPoints   []trailPoint
	splineBuf     []trailPoint
	resampleBuf   []trailPoint
	vertices      []ebiten.Vertex
	indices       []uint16
	debugVerts    []ebiten.Vertex
	debugIndices  []uint16

	segmentLifetime   float64
	totalSegmentLimit int
	currentTime       float64
	// StepDistance specifies the minimum distance between points after resampling.
	StepDistance    float64
	splinePrecision int
	jointMode       LineJointMode
	roundPrecision  int
	sharpLimit      float64
	textureMode     LineTextureMode
	tileAspect      float64
	textureScale    float64
	textureOffset   float64
	catmullRom      bool
	debugDraw       bool
}

// NewRibbonTrails creates and initializes a new RibbonTrails object with default settings.
func NewRibbonTrails() *RibbonTrails {
	return &RibbonTrails{
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
		tileAspect:        1.0,
		textureScale:      1.0,
		whiteDot:          defaultWhiteImage(),
	}
}

// SetLifetime sets the maximum time a trail segment can exist before it is removed.
func (rt *RibbonTrails) SetLifetime(lifetime float64) *RibbonTrails {
	if lifetime <= 0 {
		rt.segmentLifetime = math.MaxFloat64
	} else {
		rt.segmentLifetime = lifetime
	}
	return rt
}

// SetWidths defines the widths to interpolate across the trail's length.
func (rt *RibbonTrails) SetWidths(widths ...float64) *RibbonTrails {
	rt.segmentWidths = widths
	return rt
}

// SetColors defines the colors to interpolate across the trail's length.
func (rt *RibbonTrails) SetColors(colors ...color.RGBA) *RibbonTrails {
	rt.segmentColors = colors
	return rt
}

// SetLimit sets the maximum number of points in the trail.
// A value of 0 means there is no limit.
func (rt *RibbonTrails) SetLimit(limit int) *RibbonTrails {
	if limit < 0 {
		limit = 0
	}
	rt.totalSegmentLimit = limit
	if rt.totalSegmentLimit > 0 && len(rt.points) > rt.totalSegmentLimit {
		excess := len(rt.points) - rt.totalSegmentLimit
		copy(rt.points, rt.points[excess:])
		rt.points = rt.points[:rt.totalSegmentLimit]
	}
	return rt
}

// SetJointMode sets the style of the line joints (bevel, round, or sharp).
func (rt *RibbonTrails) SetJointMode(mode LineJointMode) *RibbonTrails {
	rt.jointMode = mode
	return rt
}

// SetTexture assigns a texture to the ribbon trail and updates the tile aspect ratio.
// Passing nil disables texturing and falls back to the solid white image.
func (rt *RibbonTrails) SetTexture(img *ebiten.Image) *RibbonTrails {
	rt.texture = img
	if img != nil {
		bounds := img.Bounds()
		if dy := bounds.Dy(); dy > 0 {
			rt.tileAspect = float64(bounds.Dx()) / float64(dy)
		}
	}
	return rt
}

// SetTextureMode defines how textures are applied along the trail.
func (rt *RibbonTrails) SetTextureMode(mode LineTextureMode) *RibbonTrails {
	rt.textureMode = mode
	return rt
}

// SetTextureScale sets the horizontal tiling scale multiplier for textured trails.
// Values <= 0 are clamped to 1e-4.
func (rt *RibbonTrails) SetTextureScale(scale float64) *RibbonTrails {
	if scale <= 0 {
		scale = 1e-4
	}
	rt.textureScale = scale
	return rt
}

// SetTextureOffset sets the horizontal UV offset (in normalized texture widths) along the trail.
func (rt *RibbonTrails) SetTextureOffset(offset float64) *RibbonTrails {
	rt.textureOffset = offset
	return rt
}

// SetStepDistance sets the minimum distance between points after resampling.
// This is useful for creating a uniform-looking trail.
func (rt *RibbonTrails) SetStepDistance(distance float64) *RibbonTrails {
	rt.StepDistance = distance
	return rt
}

// SetDebugDraw toggles the drawing of debug points and lines.
func (rt *RibbonTrails) SetDebugDraw(enabled bool) *RibbonTrails {
	rt.debugDraw = enabled
	return rt
}

// EnableCatmullRom enables or disables Catmull-Rom spline interpolation.
// The precision parameter controls the smoothness of the spline.
func (rt *RibbonTrails) EnableCatmullRom(enabled bool, precision int) *RibbonTrails {
	rt.catmullRom = enabled
	if precision > 0 {
		rt.splinePrecision = precision
	}
	return rt
}

// AddPoint adds a new point to the trail at the current time.
// It ignores duplicate points and enforces the segment limit in-place.
func (rt *RibbonTrails) AddPoint(pos Vector) {
	if len(rt.points) > 0 && rt.points[len(rt.points)-1].pos.Equals(pos) {
		return
	}
	if rt.totalSegmentLimit > 0 && len(rt.points) >= rt.totalSegmentLimit {
		excess := len(rt.points) - rt.totalSegmentLimit + 1
		copy(rt.points, rt.points[excess:])
		rt.points = rt.points[:len(rt.points)-excess]
	}
	rt.points = append(rt.points, trailPoint{pos: pos, creationTime: rt.currentTime})
}

// Update progresses the trail's simulation, removing old points and generating the mesh.
func (rt *RibbonTrails) Update(deltaTime float64) {
	rt.currentTime += deltaTime
	if rt.segmentLifetime != math.MaxFloat64 {
		alivePoints := rt.points[:0]
		for _, p := range rt.points {
			if rt.currentTime-p.creationTime < rt.segmentLifetime {
				alivePoints = append(alivePoints, p)
			}
		}
		rt.points = alivePoints
	}

	pointsToProcess := rt.points
	if rt.catmullRom && len(rt.points) >= 4 {
		pointsToProcess = rt.generateSplinePoints(rt.points, rt.splinePrecision)
	}
	if rt.StepDistance > 0 && len(pointsToProcess) > 1 {
		pointsToProcess = rt.resamplePoints(pointsToProcess, rt.StepDistance)
	}
	rt.debugPoints = pointsToProcess

	rt.vertices = rt.vertices[:0]
	rt.indices = rt.indices[:0]
	if len(rt.debugPoints) < 2 {
		return
	}

	n := len(rt.debugPoints)
	roundPrec := rt.roundPrecision
	if roundPrec <= 0 {
		roundPrec = 8
	}
	angleStep := math.Pi / float64(roundPrec)
	hasFiniteLifetime := rt.segmentLifetime != math.MaxFloat64
	invSpan := 1.0 / float64(n-1)

	textured := rt.texture != nil && rt.textureMode != LineTextureNone
	var texMinX, texW float64
	var vTop, vBot, vCenter float32
	if textured {
		texBounds := rt.texture.Bounds()
		if texBounds.Empty() {
			textured = false
		} else {
			texMinX = float64(texBounds.Min.X)
			texW = float64(texBounds.Dx())
			texMinY := float64(texBounds.Min.Y)
			texH := float64(texBounds.Dy())
			if texH <= 1 {
				vCenter = float32(texMinY + texH*0.5)
				vTop = vCenter
				vBot = vCenter
			} else {
				vTop = float32(texMinY + 0.5)
				vBot = float32(float64(texBounds.Max.Y) - 0.5)
				vCenter = (vTop + vBot) * 0.5
			}
		}
	}

	var totalDist float64
	if textured && rt.textureMode == LineTextureStretch {
		for i := 0; i < n-1; i++ {
			d := rt.debugPoints[i].pos.DistanceTo(rt.debugPoints[i+1].pos)
			if d >= 1e-6 {
				totalDist += d
			}
		}
	}
	var currentDist float64
	currentU := texMinX + rt.textureOffset*texW
	texScale := rt.textureScale
	if texScale <= 0 {
		texScale = 1.0
	}
	tileAspect := rt.tileAspect
	if tileAspect <= 0 {
		tileAspect = 1.0
	}

	for i := 0; i < n-1; i++ {
		pA := rt.debugPoints[i]
		pB := rt.debugPoints[i+1]

		var progressA, progressB float64
		if hasFiniteLifetime {
			progressA = (rt.currentTime - pA.creationTime) / rt.segmentLifetime
			progressB = (rt.currentTime - pB.creationTime) / rt.segmentLifetime
		} else {
			progressA = float64(i) * invSpan
			progressB = float64(i+1) * invSpan
		}

		colorA := interpolateRibbonColor(rt.segmentColors, progressA)
		colorB := interpolateRibbonColor(rt.segmentColors, progressB)
		widthA := interpolateRibbonWidth(rt.segmentWidths, progressA) / 2.0
		widthB := interpolateRibbonWidth(rt.segmentWidths, progressB) / 2.0

		dirAB := pB.pos.Sub(pA.pos)
		lenAB := dirAB.Length()
		if lenAB < 1e-6 {
			continue
		}

		var uA, uB float32
		if textured {
			if rt.textureMode == LineTextureStretch {
				tA := 0.0
				tB := 0.0
				if totalDist > 0 {
					tA = currentDist / totalDist
					currentDist += lenAB
					tB = currentDist / totalDist
				}
				uA = float32(texMinX + (tA+rt.textureOffset)*texW)
				uB = float32(texMinX + (tB+rt.textureOffset)*texW)
			} else {
				segWidth := widthA + widthB
				du := (lenAB / math.Max(1e-4, segWidth*tileAspect)) * texW * texScale
				uA = float32(currentU)
				currentU += du
				uB = float32(currentU)
			}
		}

		normal := dirAB.Normalize().Orthogonal()
		vA1 := pA.pos.Add(normal.ScaleF(widthA))
		vA2 := pA.pos.Sub(normal.ScaleF(widthA))
		vB1 := pB.pos.Add(normal.ScaleF(widthB))
		vB2 := pB.pos.Sub(normal.ScaleF(widthB))
		rt.addTriangle(vA1, vA2, vB1, colorA, colorA, colorB, uA, vTop, uA, vBot, uB, vTop)
		rt.addTriangle(vA2, vB2, vB1, colorA, colorB, colorB, uA, vBot, uB, vBot, uB, vTop)

		if i >= n-2 {
			continue
		}
		pC := rt.debugPoints[i+2]
		dirBC := pC.pos.Sub(pB.pos)
		if dirBC.Length() < 1e-6 {
			continue
		}
		normalB := dirBC.Normalize().Orthogonal()
		zCross := dirAB.Cross(dirBC)
		if math.Abs(zCross) < 1e-6 {
			continue
		}

		vB1Next := pB.pos.Add(normalB.ScaleF(widthB))
		vB2Next := pB.pos.Sub(normalB.ScaleF(widthB))
		switch rt.jointMode {
		case LineJointBevel:
			if zCross < 0 {
				rt.addTriangle(pB.pos, vB1, vB1Next, colorB, colorB, colorB, uB, vCenter, uB, vTop, uB, vTop)
			} else {
				rt.addTriangle(pB.pos, vB2, vB2Next, colorB, colorB, colorB, uB, vCenter, uB, vBot, uB, vBot)
			}
		case LineJointRound:
			vStart, vEnd := vB2, vB2Next
			vEdge := vBot
			if zCross < 0 {
				vStart, vEnd = vB1, vB1Next
				vEdge = vTop
			}
			v1 := vStart.Sub(pB.pos)
			v2 := vEnd.Sub(pB.pos)
			angle := math.Atan2(v1.Cross(v2), v1.Dot(v2))
			if math.Abs(angle) < angleStep {
				rt.addTriangle(pB.pos, vStart, vEnd, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			} else {
				rt.addArc(pB.pos, v1, angle, colorB, uB, vCenter, vEdge)
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
			dist := pB.pos.DistanceTo(miter)
			if widthB <= 0 || (dist/widthB) > rt.sharpLimit {
				rt.addTriangle(pB.pos, vBOuter, vBOuterNext, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			} else {
				rt.addTriangle(pB.pos, vBOuter, miter, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
				rt.addTriangle(pB.pos, miter, vBOuterNext, colorB, colorB, colorB, uB, vCenter, uB, vEdge, uB, vEdge)
			}
		}
	}
}

func (rt *RibbonTrails) generateSplinePoints(points []trailPoint, precision int) []trailPoint {
	if len(points) < 2 || precision <= 0 {
		return points
	}
	rt.splineBuf = rt.splineBuf[:0]
	rt.splineBuf = append(rt.splineBuf, points[0])
	for i := 0; i < len(points)-1; i++ {
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
			newPos := catmullRomPoint(p0.pos, p1.pos, p2.pos, p3.pos, t)
			newTime := p1.creationTime + (p2.creationTime-p1.creationTime)*t
			rt.splineBuf = append(rt.splineBuf, trailPoint{pos: newPos, creationTime: newTime})
		}
	}
	return rt.splineBuf
}

func (rt *RibbonTrails) resamplePoints(points []trailPoint, step float64) []trailPoint {
	if len(points) < 2 || step <= 0 {
		return points
	}
	rt.resampleBuf = rt.resampleBuf[:0]
	rt.resampleBuf = append(rt.resampleBuf, points[0])
	distanceNeeded := step
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]
		segmentLen := p2.pos.Sub(p1.pos).Length()
		if segmentLen < 1e-6 {
			continue
		}
		for segmentLen >= distanceNeeded {
			t := distanceNeeded / segmentLen
			newPos := p1.pos.Lerp(p2.pos, t)
			newTime := p1.creationTime + (p2.creationTime-p1.creationTime)*t
			pt := trailPoint{pos: newPos, creationTime: newTime}
			rt.resampleBuf = append(rt.resampleBuf, pt)
			segmentLen -= distanceNeeded
			p1 = pt
			distanceNeeded = step
		}
		distanceNeeded -= segmentLen
	}
	return rt.resampleBuf
}

// Draw renders the trail's mesh to the screen.
// It also draws debug visualizations if enabled.
func (rt *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if rt.whiteDot == nil {
		rt.whiteDot = defaultWhiteImage()
	}
	if len(rt.vertices) > 0 {
		var drawOp ebiten.DrawTrianglesOptions
		if op != nil {
			drawOp = *op
		}
		texture := rt.whiteDot
		if rt.texture != nil && rt.textureMode != LineTextureNone {
			texture = rt.texture
			if drawOp.Address == ebiten.AddressUnsafe && rt.textureMode == LineTextureTile {
				drawOp.Address = ebiten.AddressRepeat
			}
		}
		screen.DrawTriangles(rt.vertices, rt.indices, texture, &drawOp)
	}
	if rt.debugDraw && len(rt.debugPoints) > 0 {
		rt.debugVerts = rt.debugVerts[:0]
		rt.debugIndices = rt.debugIndices[:0]
		red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
		yellow := color.RGBA{R: 255, G: 255, B: 0, A: 255}

		for _, p := range rt.debugPoints {
			v1 := p.pos.Add(V(-2, -2))
			v2 := p.pos.Add(V(2, -2))
			v3 := p.pos.Add(V(-2, 2))
			v4 := p.pos.Add(V(2, 2))
			rt.appendDebugQuad(v1, v2, v3, v4, red)
		}
		for i := 0; i < len(rt.debugPoints)-1; i++ {
			p1 := rt.debugPoints[i].pos
			p2 := rt.debugPoints[i+1].pos
			dir := p2.Sub(p1).Normalize()
			normal := dir.Orthogonal().ScaleF(0.5)
			v1 := p1.Add(normal)
			v2 := p1.Sub(normal)
			v3 := p2.Add(normal)
			v4 := p2.Sub(normal)
			rt.appendDebugQuad(v1, v2, v3, v4, yellow)
		}
		if len(rt.debugVerts) > 0 {
			screen.DrawTriangles(rt.debugVerts, rt.debugIndices, rt.whiteDot, op)
		}
	}
}

func (rt *RibbonTrails) appendDebugQuad(v1, v2, v3, v4 Vector, clr color.RGBA) {
	const inv255 = float32(1.0 / 255.0)
	cr := float32(clr.R) * inv255
	cg := float32(clr.G) * inv255
	cb := float32(clr.B) * inv255
	ca := float32(clr.A) * inv255
	idx := uint16(len(rt.debugVerts))
	rt.debugVerts = append(rt.debugVerts,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	rt.debugIndices = append(rt.debugIndices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

func (rt *RibbonTrails) addTriangle(v1, v2, v3 Vector, c1, c2, c3 color.RGBA, u1, sv1, u2, sv2, u3, sv3 float32) {
	const inv255 = float32(1.0 / 255.0)
	idx := uint16(len(rt.vertices))
	rt.vertices = append(rt.vertices,
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
	rt.indices = append(rt.indices, idx, idx+1, idx+2)
}

func (rt *RibbonTrails) addArc(center, vbegin Vector, angleDelta float64, clr color.RGBA, u, vCenter, vEdge float32) {
	const inv255 = float32(1.0 / 255.0)
	prec := rt.roundPrecision
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

	vi := len(rt.vertices)
	rt.vertices = append(rt.vertices, ebiten.Vertex{
		DstX: float32(center.X), DstY: float32(center.Y),
		SrcX: u, SrcY: vCenter,
		ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
	})
	for i := 0; i <= steps; i++ {
		angle := t + angleStep*float64(i)
		if i == steps {
			angle = t + angleDelta
		}
		rpos := center.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(radius))
		rt.vertices = append(rt.vertices, ebiten.Vertex{
			DstX: float32(rpos.X), DstY: float32(rpos.Y),
			SrcX: u, SrcY: vEdge,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})
	}
	for i := range steps {
		rt.indices = append(rt.indices, uint16(vi), uint16(vi+i+1), uint16(vi+i+2))
	}
}

func interpolateRibbonWidth(values []float64, progress float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}
	progress = Clamp(progress, 0.0, 1.0)
	idx := progress * float64(len(values)-1)
	i1 := int(idx)
	i2 := min(i1+1, len(values)-1)
	t := idx - float64(i1)
	return values[i1]*(1-t) + values[i2]*t
}

func interpolateRibbonColor(values []color.RGBA, progress float64) color.RGBA {
	if len(values) == 0 {
		return color.RGBA{}
	}
	if len(values) == 1 {
		return values[0]
	}
	progress = Clamp(progress, 0.0, 1.0)
	idx := progress * float64(len(values)-1)
	i1 := int(idx)
	i2 := min(i1+1, len(values)-1)
	t := idx - float64(i1)
	rVal := float64(values[i1].R)*(1-t) + float64(values[i2].R)*t
	gVal := float64(values[i1].G)*(1-t) + float64(values[i2].G)*t
	bVal := float64(values[i1].B)*(1-t) + float64(values[i2].B)*t
	aVal := float64(values[i1].A)*(1-t) + float64(values[i2].A)*t
	return color.RGBA{R: uint8(rVal), G: uint8(gVal), B: uint8(bVal), A: uint8(aVal)}
}
