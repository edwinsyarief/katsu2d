package katsu2d

import "math"

type Rectangle struct {
	Min   Vector
	Max   Vector
	Angle float64
}

// NewRectangle creates a new axis-aligned rectangle.
func NewRectangle(x1, y1, x2, y2 float64) Rectangle {
	return Rectangle{
		Min: V(x1, y1),
		Max: V(x2, y2),
	}
}

// SetAngle updates the rotation angle of the rectangle.
func (self *Rectangle) SetAngle(angle float64) {
	self.Angle = angle
}

// Width returns the width of the rectangle.
func (self Rectangle) Width() float64 {
	return self.Max.X - self.Min.X
}

// Height returns the height of the rectangle.
func (self Rectangle) Height() float64 {
	return self.Max.Y - self.Min.Y
}

// Center calculates and returns the center point of the rectangle.
func (self Rectangle) Center() Vector {
	return V((self.Min.X+self.Max.X)/2, (self.Min.Y+self.Max.Y)/2)
}

// X1 returns the min X coordinate.
func (self Rectangle) X1() float64 { return self.Min.X }

// Y1 returns the min Y coordinate.
func (self Rectangle) Y1() float64 { return self.Min.Y }

// X2 returns the max X coordinate.
func (self Rectangle) X2() float64 { return self.Max.X }

// Y2 returns the max Y coordinate.
func (self Rectangle) Y2() float64 { return self.Max.Y }

// IsEmpty checks if the rectangle has no area.
func (self Rectangle) IsEmpty() bool {
	return self.Min.X >= self.Max.X || self.Min.Y >= self.Max.Y
}

// Equals checks if two rectangles are equal.
func (self Rectangle) Equals(other Rectangle) bool {
	return self.Min.Equals(other.Min) && self.Max.Equals(other.Max)
}

// Contains checks if a point is within the rectangle.
func (self Rectangle) Contains(p Vector) bool {
	return self.Min.X <= p.X && p.X < self.Max.X &&
		self.Min.Y <= p.Y && p.Y < self.Max.Y
}

// ContainsRect checks if one rectangle is completely inside another.
func (self Rectangle) ContainsRect(other Rectangle) bool {
	return self.X1() <= other.X1() &&
		self.Y1() <= other.Y1() &&
		self.X2() >= other.X2() &&
		self.Y2() >= other.Y2()
}

// Intersects checks if two rectangles intersect using the Separating Axis Theorem.
// This handles rotated rectangles.
func (self Rectangle) Intersects(other Rectangle) bool {
	// Optimization for axis-aligned rectangles.
	if self.Angle == 0 && other.Angle == 0 {
		return !self.IsEmpty() && !other.IsEmpty() &&
			self.Min.X < other.Max.X && other.Min.X < self.Max.X &&
			self.Min.Y < other.Max.Y && other.Min.Y < self.Max.Y
	}

	// Get the four axes to check for separation.
	// The axes are the normalized perpendicular vectors to the sides of each rectangle.
	axes := []Vector{
		self.GetAxis(self.Angle, 0), self.GetAxis(self.Angle, 1),
		other.GetAxis(other.Angle, 0), other.GetAxis(other.Angle, 1),
	}

	// If a separating axis is found, there is no intersection.
	for _, axis := range axes {
		if !self.OverlapOnAxis(other, axis) {
			return false
		}
	}
	return true
}

// IntersectsCircle checks if the rectangle intersects with a circle.
func (self Rectangle) IntersectsCircle(center Vector, radius float64) bool {
	closestX := math.Max(self.Min.X, math.Min(center.X, self.Max.X))
	closestY := math.Max(self.Min.Y, math.Min(center.Y, self.Max.Y))
	dx := center.X - closestX
	dy := center.Y - closestY
	return dx*dx+dy*dy <= radius*radius
}

// GetAxis returns one of the two axes of the rectangle based on its angle.
func (self Rectangle) GetAxis(angle float64, index int) Vector {
	if index == 0 {
		return Vector{math.Cos(angle), math.Sin(angle)}
	}
	return Vector{-math.Sin(angle), math.Cos(angle)}
}

// GetCorners returns the four corners of the rectangle, considering rotation.
func (self Rectangle) GetCorners() [4]Vector {
	center := self.Center()
	halfWidth := self.Width() / 2
	halfHeight := self.Height() / 2
	cos, sin := math.Cos(self.Angle), math.Sin(self.Angle)

	return [4]Vector{
		{center.X + halfWidth*cos - halfHeight*sin, center.Y + halfWidth*sin + halfHeight*cos},
		{center.X - halfWidth*cos - halfHeight*sin, center.Y - halfWidth*sin + halfHeight*cos},
		{center.X - halfWidth*cos + halfHeight*sin, center.Y - halfWidth*sin - halfHeight*cos},
		{center.X + halfWidth*cos + halfHeight*sin, center.Y + halfWidth*sin - halfHeight*cos},
	}
}

// OverlapOnAxis checks if there's overlap along a specific axis.
func (self Rectangle) OverlapOnAxis(other Rectangle, axis Vector) bool {
	proj1 := self.ProjectOntoAxis(axis)
	proj2 := other.ProjectOntoAxis(axis)
	return proj1.Min <= proj2.Max && proj2.Min <= proj1.Max
}

// ProjectOntoAxis projects the rectangle onto an axis, returning the min and max values.
func (self Rectangle) ProjectOntoAxis(axis Vector) struct{ Min, Max float64 } {
	corners := self.GetCorners()
	min := corners[0].Dot(axis)
	max := min

	for i := 1; i < 4; i++ {
		p := corners[i].Dot(axis)
		if p < min {
			min = p
		} else if p > max {
			max = p
		}
	}

	return struct{ Min, Max float64 }{min, max}
}
