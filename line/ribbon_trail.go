package line

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

type TrailScaleMode int

const (
	ScaleNone TrailScaleMode = iota
	ScaleIn
	ScaleOut
)

type TrailFadeMode int

const (
	FadeNone TrailFadeMode = iota
	FadeIn
	FadeOut
)

// RibbonTrails represents a dynamic trail rendered as a ribbon line with animation effects.
type RibbonTrails struct {
	points         []ebimath.Vector
	ages           []float64
	maxLife        float64
	width          float64
	widths         []float64
	defaultColor   color.RGBA
	colors         []color.RGBA
	startScaleMode TrailScaleMode
	startFadeMode  TrailFadeMode
	endScaleMode   TrailScaleMode
	endFadeMode    TrailFadeMode
	lineBuilder    *LineBuilder
	vertices       []ebiten.Vertex
	indices        []uint16
	whiteDot       *ebiten.Image
}

// NewRibbonTrails creates a new RibbonTrails instance.
func NewRibbonTrails() *RibbonTrails {
	r := &RibbonTrails{
		maxLife:      1.0,
		width:        10.0,
		defaultColor: color.RGBA{R: 255, G: 0, B: 0, A: 255},
		lineBuilder:  NewLineBuilder(),
		whiteDot:     ebiten.NewImage(1, 1),
	}
	r.whiteDot.Fill(color.White)
	r.lineBuilder.beginCapMode = LineCapNone
	r.lineBuilder.endCapMode = LineCapNone
	r.lineBuilder.closed = false
	r.lineBuilder.jointMode = LineJointSharp
	return r
}

// SetCapMode sets the begin cap and end cap
func (self *RibbonTrails) SetCapMode(begin, end LineCapMode) {
	self.lineBuilder.beginCapMode = begin
	self.lineBuilder.endCapMode = end
}

// SetJointMode sets the joint mode
func (self *RibbonTrails) SetJointMode(mode LineJointMode) {
	self.lineBuilder.jointMode = mode
}

// SetMaxLife sets the maximum life of trail segments in seconds.
func (self *RibbonTrails) SetMaxLife(life float64) {
	self.maxLife = life
}

// SetWidth sets the uniform width of the trail.
func (self *RibbonTrails) SetWidth(w float64) {
	self.width = w
}

// SetWidths sets the width multipliers along the normalized trail length.
func (self *RibbonTrails) SetWidths(ws ...float64) {
	self.widths = ws
}

// SetDefaultColor sets the uniform color of the trail.
func (self *RibbonTrails) SetDefaultColor(c color.RGBA) {
	self.defaultColor = c
}

// SetColors sets the colors along the normalized trail length.
func (self *RibbonTrails) SetColors(cs ...color.RGBA) {
	self.colors = cs
}

// SetStartScaleMode sets the scale mode for the start (head) of the trail.
func (self *RibbonTrails) SetStartScaleMode(mode TrailScaleMode) {
	self.startScaleMode = mode
}

// SetStartFadeMode sets the fade mode for the start (head) of the trail.
func (self *RibbonTrails) SetStartFadeMode(mode TrailFadeMode) {
	self.startFadeMode = mode
}

// SetEndScaleMode sets the scale mode for the end (tail) of the trail.
func (self *RibbonTrails) SetEndScaleMode(mode TrailScaleMode) {
	self.endScaleMode = mode
}

// SetEndFadeMode sets the fade mode for the end (tail) of the trail.
func (self *RibbonTrails) SetEndFadeMode(mode TrailFadeMode) {
	self.endFadeMode = mode
}

// Emit adds a new position to the trail head.
func (self *RibbonTrails) Emit(pos ebimath.Vector) {
	self.points = append(self.points, pos)
	self.ages = append(self.ages, 0.0)
}

// Update advances the ages of the trail segments and removes expired ones.
func (self *RibbonTrails) Update(dt float64) {
	for i := range self.ages {
		self.ages[i] += dt
	}
	for len(self.ages) > 0 && self.ages[0] > self.maxLife {
		self.points = self.points[1:]
		self.ages = self.ages[1:]
	}
	self.buildMesh()
}

func (self *RibbonTrails) buildMesh() {
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	if len(self.points) < 2 {
		return
	}

	// Compute cumulative distances for normalized t
	n := len(self.points)
	var totalDistance float64
	dists := make([]float64, n)
	for i := 1; i < n; i++ {
		d := self.points[i-1].DistanceTo(self.points[i])
		dists[i] = dists[i-1] + d
		totalDistance += d
	}
	if totalDistance < 1e-6 {
		return
	}

	// Compute normalized ages
	normAges := make([]float64, n)
	for i := range self.ages {
		normAges[i] = self.ages[i] / self.maxLife
		if normAges[i] > 1 {
			normAges[i] = 1
		}
	}

	// Compute effective widths and colors per point
	effWidths := make([]float64, n)
	effColors := make([]color.RGBA, n)
	for i := 0; i < n; i++ {
		t := dists[i] / totalDistance
		baseW := self.width * self.lerpWidth(t)
		baseC := self.lerpColor(t)

		norm := normAges[i]

		startScale := 1.0
		switch self.startScaleMode {
		case ScaleIn:
			startScale = norm
		case ScaleOut:
			startScale = 1 - norm
		}

		endScale := 1.0
		switch self.endScaleMode {
		case ScaleIn:
			endScale = norm
		case ScaleOut:
			endScale = 1 - norm
		}

		scaleM := startScale * endScale
		effWidths[i] = baseW * scaleM

		startFade := 1.0
		switch self.startFadeMode {
		case FadeIn:
			startFade = norm
		case FadeOut:
			startFade = 1 - norm
		}

		endFade := 1.0
		switch self.endFadeMode {
		case FadeIn:
			endFade = norm
		case FadeOut:
			endFade = 1 - norm
		}

		fadeM := startFade * endFade
		red, green, blue, alpha := baseC.R, baseC.G, baseC.B, baseC.A
		effColors[i] = color.RGBA{R: red, G: green, B: blue, A: uint8(float64(alpha)*fadeM + 0.5)}
	}

	// Build the line
	self.lineBuilder.vertices = self.lineBuilder.vertices[:0]
	self.lineBuilder.indices = self.lineBuilder.indices[:0]
	self.lineBuilder.points = append([]ebimath.Vector(nil), self.points...)
	self.lineBuilder.widths = effWidths
	self.lineBuilder.colors = effColors
	self.lineBuilder.Build()
	self.vertices = append([]ebiten.Vertex(nil), self.lineBuilder.vertices...)
	self.indices = append([]uint16(nil), self.lineBuilder.indices...)
}

// lerpWidth interpolates width multipliers.
func (self *RibbonTrails) lerpWidth(t float64) float64 {
	if len(self.widths) < 2 {
		return 1.0
	}
	pos := t * float64(len(self.widths)-1)
	idx1 := int(pos)
	idx2 := idx1 + 1
	if idx2 >= len(self.widths) {
		return self.widths[len(self.widths)-1]
	}
	if idx1 < 0 {
		return self.widths[0]
	}
	localT := pos - float64(idx1)
	return self.widths[idx1] + (self.widths[idx2]-self.widths[idx1])*localT
}

// lerpColor interpolates colors.
func (self *RibbonTrails) lerpColor(t float64) color.RGBA {
	if len(self.colors) == 0 {
		return self.defaultColor
	}
	if len(self.colors) == 1 {
		return self.colors[0]
	}
	pos := t * float64(len(self.colors)-1)
	segment := int(pos)
	tInSegment := pos - float64(segment)
	if segment >= len(self.colors)-1 {
		segment = len(self.colors) - 2
		tInSegment = 1.0
	}
	if segment < 0 {
		segment = 0
		tInSegment = 0.0
	}
	return utils.LerpPremultipliedRGBA(self.colors[segment], self.colors[segment+1], tInSegment)
}

// Draw renders the trail.
func (self *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(self.vertices) == 0 {
		return
	}
	screen.DrawTriangles(self.vertices, self.indices, self.whiteDot, op)
}
