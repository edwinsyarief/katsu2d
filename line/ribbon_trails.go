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
// The visual appearance of the trail is controlled entirely by the slices of
// widths and colors provided via SetWidths and SetColors.
type RibbonTrails struct {
	line              *Line
	points            []*trailPoint
	SegmentLifetime   float64
	SegmentWidths     []float64
	SegmentColors     []color.RGBA
	TotalSegmentLimit int
	currentTime       float64
}

// NewRibbonTrails creates and initializes a new RibbonTrails object with default values.
func NewRibbonTrails() *RibbonTrails {
	return &RibbonTrails{
		line:              NewLine(),
		points:            make([]*trailPoint, 0),
		SegmentLifetime:   math.MaxFloat64,                                // Default to infinite lifetime
		SegmentWidths:     []float64{10.0},                                // Default to a constant width of 10
		SegmentColors:     []color.RGBA{{R: 255, G: 255, B: 255, A: 255}}, // Default to white
		TotalSegmentLimit: 0,                                              // Unlimited
	}
}

// SetLifetime sets the lifetime of each segment in seconds.
func (self *RibbonTrails) SetLifetime(lifetime float64) *RibbonTrails {
	if lifetime <= 0 {
		self.SegmentLifetime = math.MaxFloat64
	} else {
		self.SegmentLifetime = lifetime
	}
	return self
}

// SetWidths sets the widths to be interpolated along the trail.
// To create a fade-in/out effect on the width, provide a slice like `[]float64{0, 10, 0}`.
func (self *RibbonTrails) SetWidths(widths ...float64) *RibbonTrails {
	self.SegmentWidths = widths
	return self
}

// SetColors sets the colors to be interpolated along the trail.
// To create a fade-in/out effect on the alpha, modulate the alpha component of the colors in the slice.
func (self *RibbonTrails) SetColors(colors ...color.RGBA) *RibbonTrails {
	self.SegmentColors = colors
	return self
}

// SetLimit sets the maximum number of segments in the trail.
func (self *RibbonTrails) SetLimit(limit int) *RibbonTrails {
	self.TotalSegmentLimit = limit
	return self
}

// SetJointMode defines how line segments are joined together.
func (self *RibbonTrails) SetJointMode(mode LineJointMode) *RibbonTrails {
	self.line.SetJointMode(mode)
	return self
}

// AddPoint adds a new point to the trail's head.
func (self *RibbonTrails) AddPoint(pos ebimath.Vector) {
	if self.TotalSegmentLimit > 0 && len(self.points) >= self.TotalSegmentLimit {
		self.points = self.points[1:]
	}
	self.points = append(self.points, &trailPoint{pos: pos, creationTime: self.currentTime})
}

// Update updates the state of the ribbon trail.
func (self *RibbonTrails) Update(deltaTime float64) {
	self.currentTime += deltaTime

	// Remove old points from the tail of the trail.
	if self.SegmentLifetime != math.MaxFloat64 {
		alivePoints := self.points[:0]
		for _, p := range self.points {
			if self.currentTime-p.creationTime < self.SegmentLifetime {
				alivePoints = append(alivePoints, p)
			}
		}
		self.points = alivePoints
	}

	if len(self.points) < 2 {
		self.line.ClearPoints()
		return
	}

	linePoints := make([]ebimath.Vector, len(self.points))
	for i, p := range self.points {
		linePoints[i] = p.pos
	}
	self.line.ClearPoints()
	for _, p := range linePoints {
		self.line.AddPoint(p)
	}

	numPoints := len(self.points)
	widths := make([]float64, numPoints)
	colors := make([]color.RGBA, numPoints)

	for i := 0; i < numPoints; i++ {
		// trailProgress maps from 0.0 (tail) to 1.0 (head).
		trailProgress := float64(i) / float64(numPoints-1)
		// The user expects the "start" of their config slices to be at the "head" of the trail.
		// So we invert the progress for interpolation.
		interpolationProgress := 1.0 - trailProgress

		// Interpolate base width and color from the user-provided slices.
		widths[i] = interpolateWidth(self.SegmentWidths, interpolationProgress)
		colors[i] = interpolateColor(self.SegmentColors, interpolationProgress)
	}

	self.line.SetInterpolatedWidths(widths...)
	self.line.SetInterpolatedColors(colors...)
}

// Draw renders the ribbon trail to the screen.
func (self *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	self.line.Draw(screen, op)
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
