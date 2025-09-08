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
func (r *RibbonTrails) SetLifetime(lifetime float64) *RibbonTrails {
	if lifetime <= 0 {
		r.SegmentLifetime = math.MaxFloat64
	} else {
		r.SegmentLifetime = lifetime
	}
	return r
}

// SetWidths sets the widths to be interpolated along the trail.
// To create a fade-in/out effect on the width, provide a slice like `[]float64{0, 10, 0}`.
func (r *RibbonTrails) SetWidths(widths ...float64) *RibbonTrails {
	r.SegmentWidths = widths
	return r
}

// SetColors sets the colors to be interpolated along the trail.
// To create a fade-in/out effect on the alpha, modulate the alpha component of the colors in the slice.
func (r *RibbonTrails) SetColors(colors ...color.RGBA) *RibbonTrails {
	r.SegmentColors = colors
	return r
}

// SetLimit sets the maximum number of segments in the trail.
func (r *RibbonTrails) SetLimit(limit int) *RibbonTrails {
	r.TotalSegmentLimit = limit
	return r
}

// SetJointMode defines how line segments are joined together.
func (r *RibbonTrails) SetJointMode(mode LineJointMode) *RibbonTrails {
	r.line.SetJointMode(mode)
	return r
}

// AddPoint adds a new point to the trail's head.
func (r *RibbonTrails) AddPoint(pos ebimath.Vector) {
	if r.TotalSegmentLimit > 0 && len(r.points) >= r.TotalSegmentLimit {
		r.points = r.points[1:]
	}
	r.points = append(r.points, &trailPoint{pos: pos, creationTime: r.currentTime})
}

// Update updates the state of the ribbon trail.
func (r *RibbonTrails) Update(deltaTime float64) {
	r.currentTime += deltaTime

	// Remove old points from the tail of the trail.
	if r.SegmentLifetime != math.MaxFloat64 {
		alivePoints := r.points[:0]
		for _, p := range r.points {
			if r.currentTime-p.creationTime < r.SegmentLifetime {
				alivePoints = append(alivePoints, p)
			}
		}
		r.points = alivePoints
	}

	if len(r.points) < 2 {
		r.line.ClearPoints()
		return
	}

	linePoints := make([]ebimath.Vector, len(r.points))
	for i, p := range r.points {
		linePoints[i] = p.pos
	}
	r.line.ClearPoints()
	for _, p := range linePoints {
		r.line.AddPoint(p)
	}

	numPoints := len(r.points)
	widths := make([]float64, numPoints)
	colors := make([]color.RGBA, numPoints)

	for i := 0; i < numPoints; i++ {
		// trailProgress maps from 0.0 (tail) to 1.0 (head).
		trailProgress := float64(i) / float64(numPoints-1)
		// The user expects the "start" of their config slices to be at the "head" of the trail.
		// So we invert the progress for interpolation.
		interpolationProgress := 1.0 - trailProgress

		// Interpolate base width and color from the user-provided slices.
		widths[i] = interpolateWidth(r.SegmentWidths, interpolationProgress)
		colors[i] = interpolateColor(r.SegmentColors, interpolationProgress)
	}

	r.line.SetInterpolatedWidths(widths...)
	r.line.SetInterpolatedColors(colors...)
}

// Draw renders the ribbon trail to the screen.
func (r *RibbonTrails) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	r.line.Draw(screen, op)
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
