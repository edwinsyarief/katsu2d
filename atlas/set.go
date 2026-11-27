package atlas

import (
	"image"
	"slices"
	"sort"
)

// Set manages the allocation of rectangular regions within a 2D space.
// It uses a rectangle packing algorithm to keep track of used and empty
// areas, allowing for efficient insertion and removal of rectangles.
type Set struct {
	rects   []*image.Rectangle // rects stores the currently allocated (filled) regions.
	empties []image.Rectangle  // empties stores the available (free) regions.
	tmps    []image.Rectangle  // tmps is a temporary slice used for intermediate calculations to reduce allocations.
	minSize image.Point        // minSize specifies the minimum dimensions for an empty region to be considered usable.
	width   int                // width is the total width of the area being managed.
	height  int                // height is the total height of the area being managed.
}

// NewSetOptions provides optional configuration for creating a new Set.
type NewSetOptions struct {
	// MinSize specifies the minimum size for a region to be considered for allocation.
	// Regions smaller than this will be discarded.
	MinSize image.Point
}

// NewSet creates and initializes a new Set with given dimensions and options.
func NewSet(width, height int, opts *NewSetOptions) *Set {
	s := &Set{
		width:  width,
		height: height,
		// Initially, the entire area is one large empty rectangle.
		empties: []image.Rectangle{
			image.Rect(0, 0, width, height),
		},
	}
	if opts != nil {
		s.minSize = opts.MinSize
	}
	// Ensure minSize is at least 1x1 to avoid issues with zero-sized regions.
	s.minSize.X = max(s.minSize.X, 1)
	s.minSize.Y = max(s.minSize.Y, 1)
	return s
}

// appendEmptyNeighbours calculates the new empty regions created when a smaller
// rectangle ('filled') is placed inside a larger one ('parent').
func appendEmptyNeighbours(rects []image.Rectangle, parent, filled image.Rectangle) []image.Rectangle {
	if !filled.In(parent) {
		return append(rects, parent)
	}
	// Add the area to the left of the filled rectangle.
	if filled.Min.X > parent.Min.X {
		rects = append(rects, image.Rect(
			parent.Min.X,
			parent.Min.Y,
			filled.Min.X,
			parent.Max.Y,
		))
	}
	// Add the area to the right of the filled rectangle.
	if filled.Max.X < parent.Max.X {
		rects = append(rects, image.Rect(
			filled.Max.X,
			parent.Min.Y,
			parent.Max.X,
			parent.Max.Y,
		))
	}
	// Add the area above the filled rectangle.
	if filled.Min.Y > parent.Min.Y {
		rects = append(rects, image.Rect(
			parent.Min.X,
			parent.Min.Y,
			parent.Max.X,
			filled.Min.Y,
		))
	}
	// Add the area below the filled rectangle.
	if filled.Max.Y < parent.Max.Y {
		rects = append(rects, image.Rect(
			parent.Min.X,
			filled.Max.Y,
			parent.Max.X,
			parent.Max.Y,
		))
	}
	return rects
}

// sanitizeEmptyRegions cleans up the list of empty regions. It removes regions
// that are too small and eliminates any region that is fully contained within another.
func (s *Set) sanitizeEmptyRegions(current []image.Rectangle) {
	// Sort empty regions by size (largest first) to ensure smaller, contained
	// regions are correctly identified and removed.
	sort.SliceStable(current, func(i, j int) bool {
		si := current[i].Dx() * current[i].Dy()
		sj := current[j].Dx() * current[j].Dy()
		return si > sj
	})
	s.empties = s.empties[:0]
	for i := range current {
		// Discard regions that are smaller than the configured minimum size.
		if current[i].Dx() < s.minSize.X || current[i].Dy() < s.minSize.Y {
			continue
		}
		var contained bool
		// Check if the current region is a duplicate or is contained by any
		// region already in the cleaned list.
		for _, empty := range s.empties {
			if current[i] == empty || current[i].In(empty) {
				contained = true
				break
			}
		}
		if contained {
			continue
		}
		s.empties = append(s.empties, current[i])
	}
}

// Insert finds a suitable empty region for the given rectangle, places it,
// and updates the list of empty regions. Returns true on success.
func (s *Set) Insert(rect *image.Rectangle) bool {
	// Filter out empty regions that are too small to fit the new rectangle.
	s.tmps = s.tmps[:0]
	for _, r := range s.empties {
		if rect.Dx() <= r.Dx() && rect.Dy() <= r.Dy() {
			s.tmps = append(s.tmps, r)
		}
	}
	// If no suitable region is found, insertion fails.
	if len(s.tmps) == 0 {
		return false
	}
	// Find the best-fit rectangle (closest to the top-left corner).
	// This strategy helps keep allocations packed together.
	best := s.tmps[0]
	bs := best.Min.X + best.Min.Y
	for i := range s.tmps {
		if d := s.tmps[i].Min.X + s.tmps[i].Min.Y; d < bs {
			best = s.tmps[i]
			bs = d
		}
	}
	// Position the input rectangle at the top-left corner of the best-fit empty region.
	*rect = rect.Add(best.Min)
	s.rects = append(s.rects, rect)
	// Recalculate the empty regions based on the new allocation.
	s.tmps = s.tmps[:0]
	for i := range s.empties {
		// If the new rect intersects with an empty region, split that region.
		if ix := rect.Intersect(s.empties[i]); !ix.Empty() {
			s.tmps = appendEmptyNeighbours(s.tmps, s.empties[i], ix)
		} else {
			// Otherwise, keep the empty region as is.
			s.tmps = append(s.tmps, s.empties[i])
		}
	}
	// Clean up the newly calculated empty regions for the next insertion.
	s.sanitizeEmptyRegions(s.tmps)
	return true
}

// intersectAny checks if a rectangle 'r' intersects with any rectangle in a given slice.
func intersectAny(r image.Rectangle, rectangles []image.Rectangle) bool {
	for i := range rectangles {
		if !rectangles[i].Intersect(r).Empty() {
			return true
		}
	}
	return false
}

// Free removes a rectangle from the set of allocated regions and attempts to merge
// the newly freed space with adjacent empty regions.
func (s *Set) Free(rect *image.Rectangle) {
	if rect == nil || len(s.rects) == 0 {
		return
	}
	// Find and remove the rectangle from the list of allocated rects.
	idx := slices.Index(s.rects, rect)
	if idx != -1 {
		s.rects = slices.Delete(s.rects, idx, idx+1)
		// This algorithm attempts to grow the freed region by expanding it horizontally
		// and vertically until it hits another allocated region.
		// Grow horizontally.
		pushX := func(base image.Rectangle) image.Rectangle {
			// Consider only rects that could block horizontal expansion.
			s.tmps = s.tmps[:0]
			for _, r := range s.rects {
				if r.Max.Y >= base.Min.Y && r.Min.Y <= base.Max.Y {
					s.tmps = append(s.tmps, *r)
				}
			}
			// Expand left.
			for base.Min.X > 0 {
				base.Min.X--
				if intersectAny(base, s.tmps) {
					base.Min.X++
					break
				}
			}
			// Expand right.
			for base.Max.X < s.width {
				base.Max.X++
				if intersectAny(base, s.tmps) {
					base.Max.X--
					break
				}
			}
			return base
		}
		// Grow vertically.
		pushY := func(base image.Rectangle) image.Rectangle {
			// Consider only rects that could block vertical expansion.
			s.tmps = s.tmps[:0]
			for _, r := range s.rects {
				if r.Max.X >= base.Min.X && r.Min.X <= base.Max.X {
					s.tmps = append(s.tmps, *r)
				}
			}
			// Expand up.
			for base.Min.Y > 0 {
				base.Min.Y--
				if intersectAny(base, s.tmps) {
					base.Min.Y++
					break
				}
			}
			// Expand down.
			for base.Max.Y < s.height {
				base.Max.Y++
				if intersectAny(base, s.tmps) {
					base.Max.Y--
					break
				}
			}
			return base
		}
		// Calculate the largest possible free rectangle by trying both expansion orders.
		// Order matters: expanding X then Y can result in a different shape than Y then X.
		rX := pushX(*rect)
		rX = pushY(rX)
		rY := pushY(*rect)
		rY = pushX(rY)
		// Keep the resulting rectangle with the largest area.
		sX := rX.Dx() * rX.Dy()
		sY := rY.Dx() * rY.Dy()
		var freed image.Rectangle
		if sX > sY {
			freed = rX
		} else {
			freed = rY
		}
		// Add the new, larger freed region to the list of empty regions.
		s.empties = append(s.empties, freed)
		s.tmps = append(s.tmps[:0], s.empties...)
		// Sanitize the list to merge overlapping empty regions.
		s.sanitizeEmptyRegions(s.tmps)
	}
}
