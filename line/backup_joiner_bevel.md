# Bevel join backup code

```go
package line

import (
    "image/color"
    "math"

    ebimath "github.com/edwinsyarief/ebi-math"
    "github.com/edwinsyarief/katsu2d/utils"
    "github.com/hajimehoshi/ebiten/v2"
)

// BevelJoiner handles bevel joins with optional tight-angle squeeze guard.
type BevelJoiner struct {
    // TightAngleThreshold in radians for the interior turn angle between -dir1 and dir2.
    // If <= 0, defaults to 20° (~0.349 rad).
    TightAngleThreshold float64
    // SqueezeGuardFactor ensures the inner pivot lies at least this × max(halfWidths) from the joint.
    // If <= 0, defaults to 1.0.
    SqueezeGuardFactor float64
}

// dotClamp clamps the dot product to [-1, 1] to avoid NaNs in acos.
func dotClamp(ax, ay, bx, by float64) float64 {
    d := ax*bx + ay*by
    if d > 1 {
        return 1
    }
    if d < -1 {
        return -1
    }
    return d
}

// projectParam returns the scalar projection of P onto the direction vector from A.
func projectParam(ax, ay, dirx, diry, px, py float64) float64 {
    dx := px - ax
    dy := py - ay
    return dx*dirx + dy*diry
}

// halfWidthFromPair returns half the distance between two points.
func halfWidthFromPair(ax, ay, bx, by float64) float64 {
    return 0.5 * math.Hypot(ax-bx, ay-by)
}

// createJoint computes the bevel join between two PolySegments.
func (self *BevelJoiner) createJoint(
    vertices *[]ebiten.Vertex,
    indices *[]uint16,
    segment1, segment2 PolySegment,
    col color.RGBA,
    opacity float64,
    end1, end2, nextStart1, nextStart2 *ebimath.Vector,
    addGeometry bool,
) {
    threshold := self.TightAngleThreshold
    if threshold <= 0 {
        threshold = 20.0 * math.Pi / 180.0
    }
    guard := self.SqueezeGuardFactor
    if guard <= 0 {
        guard = 1.0
    }

    dir1 := segment1.Center.Direction(true)
    dir2 := segment2.Center.Direction(true)

    clockwise := (dir1.X*dir2.Y - dir1.Y*dir2.X) < 0

    var inner1, inner2, outer1, outer2 *LineSegment
    if clockwise {
        outer1 = &segment1.Edge1
        outer2 = &segment2.Edge1
        inner1 = &segment1.Edge2
        inner2 = &segment2.Edge2
    } else {
        outer1 = &segment1.Edge2
        outer2 = &segment2.Edge2
        inner1 = &segment1.Edge1
        inner2 = &segment2.Edge1
    }

    innerSec, ok := inner1.Intersection(*inner2, false)
    if !ok {
        innerSec = inner1.B
    }

    // Interior turn angle between -dir1 and dir2
    turnAngle := math.Acos(dotClamp(-dir1.X, -dir1.Y, dir2.X, dir2.Y))

    // Tight-angle clamp
    if turnAngle < threshold {
        inner2Dir := inner2.Direction(true)
        norm := math.Hypot(inner2Dir.X, inner2Dir.Y)
        if norm > 0 {
            ux := inner2Dir.X / norm
            uy := inner2Dir.Y / norm

            // Use max half-width from both sides of the joint
            w1 := halfWidthFromPair(outer1.B.X, outer1.B.Y, inner1.B.X, inner1.B.Y)
            w2 := halfWidthFromPair(outer2.A.X, outer2.A.Y, inner2.A.X, inner2.A.Y)
            sMin := guard * math.Max(w1, w2)

            s := projectParam(inner2.A.X, inner2.A.Y, ux, uy, innerSec.X, innerSec.Y)
            if s < sMin {
                s = sMin
            }
            inner2Len := math.Hypot(inner2.B.X-inner2.A.X, inner2.B.Y-inner2.A.Y)
            if s > inner2Len {
                s = inner2Len
            }

            innerSec = ebimath.V(inner2.A.X+ux*s, inner2.A.Y+uy*s)
        } else {
            innerSec = inner2.A
        }
    }

    // Assign end/start points for main segment quads
    if clockwise {
        *end1 = outer1.B
        *end2 = innerSec
        *nextStart1 = outer2.A
        *nextStart2 = innerSec
    } else {
        *end1 = innerSec
        *end2 = outer1.B
        *nextStart1 = innerSec
        *nextStart2 = outer2.A
    }

    // Emit join wedge triangle if requested
    if addGeometry && vertices != nil && indices != nil {
        // Skip wedge if nearly straight
        if math.Abs(turnAngle-math.Pi) > 1e-3 {
            var v0, v1, v2 ebimath.Vector
            if clockwise {
                v0, v1, v2 = outer1.B, outer2.A, innerSec
            } else {
                v0, v1, v2 = innerSec, outer2.A, outer1.B
            }

            idx := uint16(len(*vertices))
            *vertices = append(*vertices,
                utils.CreateVertexWithOpacity(v0, ebimath.ZeroVector, col, opacity),
                utils.CreateVertexWithOpacity(v1, ebimath.ZeroVector, col, opacity),
                utils.CreateVertexWithOpacity(v2, ebimath.ZeroVector, col, opacity),
            )
            *indices = append(*indices, idx, idx+1, idx+2)
        }
    }
}

// addQuad appends a quad (two triangles) to the mesh.
func addQuad(
    vertices *[]ebiten.Vertex,
    indices *[]uint16,
    s1, s2, e1, e2 ebimath.Vector,
    colStart, colEnd color.RGBA,
    opacity float64,
) {
    base := uint16(len(*vertices))
    *vertices = append(*vertices,
        utils.CreateVertexWithOpacity(s1, ebimath.ZeroVector, colStart, opacity),
        utils.CreateVertexWithOpacity(s2, ebimath.ZeroVector, colStart, opacity),
        utils.CreateVertexWithOpacity(e1, ebimath.ZeroVector, colEnd, opacity),
        utils.CreateVertexWithOpacity(e2, ebimath.ZeroVector, colEnd, opacity),
    )
    *indices = append(*indices,
        base, base+1, base+2,
        base+2, base+1, base+3,
    )
}

// BuildMesh constructs the vertex/index buffers for the line with bevel joins.
func (self *BevelJoiner) BuildMesh(l *Line) ([]ebiten.Vertex, []uint16) {
    if len(l.points) < 2 {
        return nil, nil
    }

    vertices := make([]ebiten.Vertex, 0, len(l.points)*6)
    indices := make([]uint16, 0, len(l.points)*12)

    // Build PolySegments from consecutive points
    segments := make([]PolySegment, 0, len(l.points))
    for i := 0; i < len(l.points)-1; i++ {
        p1 := l.points[i]
        p2 := l.points[i+1]
        if !p1.position.Equals(p2.position) {
            segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2, p2.width/2))
        }
    }
    // Add closing segment if closed polyline
    if l.IsClosed && len(l.points) > 1 {
        p1 := l.points[len(l.points)-1]
        p2 := l.points[0]
        if !p1.position.Equals(p2.position) {
            segments = append(segments, NewPolySegment(p1.position, p2.position, p1.width/2, p2.width/2))
        }
    }

    if len(segments) == 0 {
        return nil, nil
    }

    totalSegments := len(l.points) - 1
    if l.IsClosed {
        totalSegments = len(l.points)
    }

    // Color interpolation helper
    lerpColorAt := func(t float64) color.RGBA {
        if l.interpolateColor {
            return l.lerpColor(t)
        }
        return color.RGBA{255, 255, 255, 255}
    }

    // Determine starting edge points
    var start1, start2 ebimath.Vector
    if l.IsClosed {
        firstSeg := segments[0]
        lastSeg := segments[len(segments)-1]
        seamColor := func() color.RGBA {
            if l.interpolateColor {
                return lerpColorAt(0.0)
            }
            return l.points[0].color
        }()
        var tmpEnd1, tmpEnd2 ebimath.Vector
        // Force anti-squeeze logic to run by calling createJoint normally
        self.createJoint(nil, nil, lastSeg, firstSeg, seamColor, l.opacity,
            &tmpEnd1, &tmpEnd2, &start1, &start2, false)
    } else {
        start1 = segments[0].Edge1.A
        start2 = segments[0].Edge2.A
    }

    // Main segment loop
    for i := 0; i < len(segments); i++ {
        segment := segments[i]
        isLast := i == len(segments)-1

        var p1Idx, p2Idx int
        if !l.IsClosed {
            p1Idx = i
            p2Idx = i + 1
        } else {
            p1Idx = i
            p2Idx = (i + 1) % len(l.points)
        }

        colStart := l.points[p1Idx].color
        colEnd := l.points[p2Idx].color
        if l.interpolateColor {
            colStart = lerpColorAt(float64(p1Idx) / float64(totalSegments))
            colEnd = lerpColorAt(float64(p2Idx) / float64(totalSegments))
        }

        end1 := segment.Edge1.B
        end2 := segment.Edge2.B

        if !isLast {
            // Join to next segment
            nextSeg := segments[(i+1)%len(segments)]
            var nextStart1, nextStart2 ebimath.Vector
            self.createJoint(&vertices, &indices, segment, nextSeg, colEnd, l.opacity,
                &end1, &end2, &nextStart1, &nextStart2, true)

            // Add main quad for this segment
            addQuad(&vertices, &indices, start1, start2, end1, end2, colStart, colEnd, l.opacity)

            // Prepare for next iteration
            start1 = nextStart1
            start2 = nextStart2
        } else {
            // Last segment — add quad
            addQuad(&vertices, &indices, start1, start2, end1, end2, colStart, colEnd, l.opacity)

            // If closed, connect last to first with a join
            if l.IsClosed {
                firstSeg := segments[0]
                var seamEnd1, seamEnd2, seamNext1, seamNext2 ebimath.Vector
                self.createJoint(&vertices, &indices, segment, firstSeg, colEnd, l.opacity,
                    &seamEnd1, &seamEnd2, &seamNext1, &seamNext2, true)
            }
        }
    }

    return vertices, indices
}
```
