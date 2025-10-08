package katsu2d

import (
	"fmt"
	"math"
)

// Vector represents a 2D vector with X and Y components.
type Vector struct {
	X, Y float64
}

var (
	// ZeroVector represents a Vector at the origin (0, 0).
	ZeroVector = V2(0)
	// Right represents a unit vector pointing to the right (1, 0).
	Right = V(1, 0)
	// Left represents a unit vector pointing to the left (-1, 0).
	Left = Right.Negate()
	// Up represents a unit vector pointing up (0, 1).
	Up = V(0, 1)
	// Down represents a unit vector pointing down (0, -1).
	Down = Up.Negate()
)

// V creates a new Vector with given x and y coordinates.
func V(x, y float64) Vector {
	return Vector{X: x, Y: y}
}

// VInt converts integer coordinates to a Vector.
func VInt(x, y int) Vector {
	return V(float64(x), float64(y))
}

// V2 creates a Vector where both X and Y are set to the same value.
func V2(v float64) Vector {
	return V(v, v)
}

// V2Int creates a Vector where both X and Y are set to the integer value converted to float64.
func V2Int(v int) Vector {
	return V2(float64(v))
}

// String returns a string representation of the Vector.
func (self Vector) String() string {
	return fmt.Sprintf("[%f, %f]", self.X, self.Y)
}

// IsZero checks if the Vector is at the origin (0, 0).
func (self Vector) IsZero() bool {
	return self.X == 0 && self.Y == 0
}

// IsNormalized checks if the vector's length is approximately 1.
func (self Vector) IsNormalized() bool {
	return EqualsApproximately(self.LengthSquared(), 1)
}

// Abs returns a new Vector with the absolute values of X and Y.
func (self Vector) Abs() Vector {
	return V(math.Abs(self.X), math.Abs(self.Y))
}

// ToInt converts the Vector to integer coordinates.
func (self Vector) ToInt() (int, int) {
	return int(self.X), int(self.Y)
}

// Apply applies a matrix transformation to this Vector.
func (self Vector) Apply(m Matrix) Vector {
	x, y := m.Apply(self.X, self.Y)
	return V(x, y)
}

// RotateDegrees rotates the Vector by degrees.
func (self Vector) RotateDegrees(degrees float64) Vector {
	degrees = Repeat(degrees, 360) // Normalize angle
	radians := degrees * (Pi / 180)
	return self.Rotate(radians)
}

// Rotate rotates the Vector by the given angle in radians.
func (self Vector) Rotate(angle float64) Vector {
	sine, cosine := math.Sin(angle), math.Cos(angle)
	return V(self.X*cosine-self.Y*sine, self.X*sine+self.Y*cosine)
}

// RotateAround rotates this Vector around another Vector by an angle in radians.
func (self Vector) RotateAround(around Vector, angle float64) Vector {
	return V(
		math.Cos(angle)*(self.X-around.X)-math.Sin(angle)*(self.Y-around.Y)+around.X,
		math.Sin(angle)*(self.X-around.X)+math.Cos(angle)*(self.Y-around.Y)+around.Y,
	)
}

// DistanceTo calculates the Euclidean distance to another Vector.
func (self Vector) DistanceTo(v2 Vector) float64 {
	return math.Sqrt(self.DistanceSquaredTo(v2))
}

// DistanceSquaredTo computes the squared distance to another Vector.
func (self Vector) DistanceSquaredTo(v2 Vector) float64 {
	dx, dy := self.X-v2.X, self.Y-v2.Y
	return dx*dx + dy*dy
}

// Dot computes the dot product between this Vector and another.
func (self Vector) Dot(v2 Vector) float64 {
	return self.X*v2.X + self.Y*v2.Y
}

// Length returns the magnitude of the Vector.
func (self Vector) Length() float64 {
	return math.Sqrt(self.LengthSquared())
}

// LengthSquared returns the square of the Vector's length.
func (self Vector) LengthSquared() float64 {
	return self.Dot(self)
}

// Angle returns the angle of the Vector from the positive X-axis in radians.
func (self Vector) Angle() float64 {
	return math.Atan2(self.Y, self.X)
}

// AngleToPoint returns the angle from this Vector towards another point.
func (self Vector) AngleToPoint(other Vector) float64 {
	return other.Sub(self).Angle()
}

// DirectionTo returns a normalized vector pointing from this Vector to another.
func (self Vector) DirectionTo(other Vector) Vector {
	return other.Sub(self).Normalize()
}

// VecTowards calculates a Vector of given length towards another point.
func (self Vector) VecTowards(other Vector, length float64) Vector {
	angle := self.AngleToPoint(other)
	return V(math.Cos(angle), math.Sin(angle)).ScaleF(length)
}

// MoveTowards moves the Vector towards another Vector by a maximum distance.
func (self Vector) MoveTowards(other Vector, length float64) Vector {
	direction := other.Sub(self)
	dist := direction.Length()
	if dist <= length || dist < Epsilon {
		return other
	}
	return self.Add(direction.DivF(dist).ScaleF(length))
}

// Negate returns a new Vector with both components negated.
func (self Vector) Negate() Vector {
	return V(-self.X, -self.Y)
}

// Add adds one or more Vectors to this Vector, returning a new Vector.
func (self Vector) Add(others ...Vector) Vector {
	result := self
	for _, r := range others {
		result.X += r.X
		result.Y += r.Y
	}
	return result
}

// AddF adds a scalar to both components of the Vector, returning a new Vector.
func (self Vector) AddF(scalar float64) Vector {
	return V(self.X+scalar, self.Y+scalar)
}

// Sub subtracts one or more Vectors from this Vector, returning a new Vector.
func (self Vector) Sub(others ...Vector) Vector {
	result := self
	for _, r := range others {
		result.X -= r.X
		result.Y -= r.Y
	}
	return result
}

// SubF subtracts a scalar from both components of the Vector, returning a new Vector.
func (self Vector) SubF(scalar float64) Vector {
	return V(self.X-scalar, self.Y-scalar)
}

// Div divides the Vector by another Vector component-wise, returning a new Vector.
func (self Vector) Div(other Vector) Vector {
	return V(self.X/other.X, self.Y/other.Y)
}

// DivF divides the Vector by a scalar, returning a new Vector.
func (self Vector) DivF(scalar float64) Vector {
	return V(self.X/scalar, self.Y/scalar)
}

// Scale scales the Vector by another Vector component-wise, returning a new Vector.
func (self Vector) Scale(other Vector) Vector {
	return V(self.X*other.X, self.Y*other.Y)
}

// ScaleF scales the Vector by a scalar, returning a new Vector.
func (self Vector) ScaleF(scalar float64) Vector {
	return V(self.X*scalar, self.Y*scalar)
}

// Normalize returns a unit vector in the same direction as this Vector.
func (self Vector) Normalize() Vector {
	l := self.LengthSquared()
	if l != 0 {
		return self.ScaleF(1 / math.Sqrt(l))
	}
	return self
}

// Lerp performs linear interpolation between two vectors.
// t is the interpolation factor, typically between 0 and 1.
func (v Vector) Lerp(other Vector, t float64) Vector {
	// The formula is: v + (other - v) * t
	return v.Add(other.Sub(v).ScaleF(t))
}

// ClampLength ensures the Vector's length does not exceed a given limit.
func (self Vector) ClampLength(limit float64) Vector {
	l := self.Length()
	if l > 0 && l > limit {
		return self.Normalize().ScaleF(limit)
	}
	return self
}

// Extend adds magnitude to the Vector in the direction it's already pointing.
func (self Vector) Extend(length float64) Vector {
	return self.Add(self.Normalize().ScaleF(length))
}

// Shorten subtracts the given magnitude from the Vector's existing magnitude.
func (self Vector) Shorten(limit float64) Vector {
	if self.Length() > limit {
		return self.Sub(self.Normalize().ScaleF(limit))
	}
	return Vector{0, 0}
}

// Round returns a new Vector with each component rounded to the nearest integer.
func (self Vector) Round() Vector {
	return V(math.Round(self.X), math.Round(self.Y))
}

// Floor returns a new Vector with each component rounded down to the nearest integer.
func (self Vector) Floor() Vector {
	return V(math.Floor(self.X), math.Floor(self.Y))
}

// Ceil returns a new Vector with each component rounded up to the nearest integer.
func (self Vector) Ceil() Vector {
	return V(math.Ceil(self.X), math.Ceil(self.Y))
}

// MoveInDirection moves the Vector in the direction of the angle by a given distance.
func (self Vector) MoveInDirection(angle, distance float64) Vector {
	return self.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(distance))
}

// Equals checks if two Vectors are equal within a small tolerance.
func (self Vector) Equals(other Vector) bool {
	return self.Sub(other).LengthSquared() < 0.00000000009
}

// Reflect reflects the vector against the given surface normal.
func (self Vector) Reflect(normal Vector) Vector {
	n := normal.Normalize()
	return self.Sub(n.ScaleF(2 * n.Dot(self)))
}

func (self Vector) Orthogonal() Vector {
	return V(-self.Y, self.X)
}

func (self Vector) Cross(other Vector) float64 {
	return self.X*other.Y - self.Y*other.X
}
