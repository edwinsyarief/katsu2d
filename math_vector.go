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
func (v Vector) String() string {
	return fmt.Sprintf("[%f, %f]", v.X, v.Y)
}

// IsZero checks if the Vector is at the origin (0, 0).
func (v Vector) IsZero() bool {
	return v.X == 0 && v.Y == 0
}

// IsNormalized checks if the vector's length is approximately 1.
func (v Vector) IsNormalized() bool {
	return EqualsApproximately(v.LengthSquared(), 1)
}

// Abs returns a new Vector with the absolute values of X and Y.
func (v Vector) Abs() Vector {
	return V(math.Abs(v.X), math.Abs(v.Y))
}

// ToInt converts the Vector to integer coordinates.
func (v Vector) ToInt() (int, int) {
	return int(v.X), int(v.Y)
}

// Apply applies a matrix transformation to this Vector.
func (v Vector) Apply(m Matrix) Vector {
	x, y := m.Apply(v.X, v.Y)
	return V(x, y)
}

// RotateDegrees rotates the Vector by degrees.
func (v Vector) RotateDegrees(degrees float64) Vector {
	degrees = Repeat(degrees, 360) // Normalize angle
	radians := degrees * (Pi / 180)
	return v.Rotate(radians)
}

// Rotate rotates the Vector by the given angle in radians.
func (v Vector) Rotate(angle float64) Vector {
	sine, cosine := math.Sin(angle), math.Cos(angle)
	return V(v.X*cosine-v.Y*sine, v.X*sine+v.Y*cosine)
}

// RotateAround rotates this Vector around another Vector by an angle in radians.
func (v Vector) RotateAround(around Vector, angle float64) Vector {
	return V(
		math.Cos(angle)*(v.X-around.X)-math.Sin(angle)*(v.Y-around.Y)+around.X,
		math.Sin(angle)*(v.X-around.X)+math.Cos(angle)*(v.Y-around.Y)+around.Y,
	)
}

// DistanceTo calculates the Euclidean distance to another Vector.
func (v Vector) DistanceTo(v2 Vector) float64 {
	return math.Sqrt(v.DistanceSquaredTo(v2))
}

// DistanceSquaredTo computes the squared distance to another Vector.
func (v Vector) DistanceSquaredTo(v2 Vector) float64 {
	dx, dy := v.X-v2.X, v.Y-v2.Y
	return dx*dx + dy*dy
}

// Dot computes the dot product between this Vector and another.
func (v Vector) Dot(v2 Vector) float64 {
	return v.X*v2.X + v.Y*v2.Y
}

// Length returns the magnitude of the Vector.
func (v Vector) Length() float64 {
	return math.Sqrt(v.LengthSquared())
}

// LengthSquared returns the square of the Vector's length.
func (v Vector) LengthSquared() float64 {
	return v.Dot(v)
}

// Angle returns the angle of the Vector from the positive X-axis in radians.
func (v Vector) Angle() float64 {
	return math.Atan2(v.Y, v.X)
}

// AngleToPoint returns the angle from this Vector towards another point.
func (v Vector) AngleToPoint(other Vector) float64 {
	return other.Sub(v).Angle()
}

// DirectionTo returns a normalized vector pointing from this Vector to another.
func (v Vector) DirectionTo(other Vector) Vector {
	return other.Sub(v).Normalize()
}

// VecTowards calculates a Vector of given length towards another point.
func (v Vector) VecTowards(other Vector, length float64) Vector {
	angle := v.AngleToPoint(other)
	return V(math.Cos(angle), math.Sin(angle)).ScaleF(length)
}

// MoveTowards moves the Vector towards another Vector by a maximum distance.
func (v Vector) MoveTowards(other Vector, length float64) Vector {
	direction := other.Sub(v)
	dist := direction.Length()
	if dist <= length || dist < Epsilon {
		return other
	}
	return v.Add(direction.DivF(dist).ScaleF(length))
}

// Negate returns a new Vector with both components negated.
func (v Vector) Negate() Vector {
	return V(-v.X, -v.Y)
}

// Add adds one or more Vectors to this Vector, returning a new Vector.
func (v Vector) Add(others ...Vector) Vector {
	result := v
	for _, o := range others {
		result.X += o.X
		result.Y += o.Y
	}
	return result
}

// AddF adds a scalar to both components of the Vector, returning a new Vector.
func (v Vector) AddF(scalar float64) Vector {
	return V(v.X+scalar, v.Y+scalar)
}

// Sub subtracts one or more Vectors from this Vector, returning a new Vector.
func (v Vector) Sub(others ...Vector) Vector {
	result := v
	for _, o := range others {
		result.X -= o.X
		result.Y -= o.Y
	}
	return result
}

// SubF subtracts a scalar from both components of the Vector, returning a new Vector.
func (v Vector) SubF(scalar float64) Vector {
	return V(v.X-scalar, v.Y-scalar)
}

// Div divides the Vector by another Vector component-wise, returning a new Vector.
func (v Vector) Div(other Vector) Vector {
	return V(v.X/other.X, v.Y/other.Y)
}

// DivF divides the Vector by a scalar, returning a new Vector.
func (v Vector) DivF(scalar float64) Vector {
	return V(v.X/scalar, v.Y/scalar)
}

// Scale scales the Vector by another Vector component-wise, returning a new Vector.
func (v Vector) Scale(other Vector) Vector {
	return V(v.X*other.X, v.Y*other.Y)
}

// ScaleF scales the Vector by a scalar, returning a new Vector.
func (v Vector) ScaleF(scalar float64) Vector {
	return V(v.X*scalar, v.Y*scalar)
}

// Normalize returns a unit vector in the same direction as this Vector.
func (v Vector) Normalize() Vector {
	l := v.LengthSquared()
	if l != 0 {
		return v.ScaleF(1 / math.Sqrt(l))
	}
	return v
}

// Lerp performs linear interpolation between two vectors.
// t is the interpolation factor, typically between 0 and 1.
func (v Vector) Lerp(other Vector, t float64) Vector {
	// The formula is: v + (other - v) * t
	return v.Add(other.Sub(v).ScaleF(t))
}

// ClampLength ensures the Vector's length does not exceed a given limit.
func (v Vector) ClampLength(limit float64) Vector {
	l := v.Length()
	if l > 0 && l > limit {
		return v.Normalize().ScaleF(limit)
	}
	return v
}

// Extend adds magnitude to the Vector in the direction it's already pointing.
func (v Vector) Extend(length float64) Vector {
	return v.Add(v.Normalize().ScaleF(length))
}

// Shorten subtracts the given magnitude from the Vector's existing magnitude.
func (v Vector) Shorten(limit float64) Vector {
	if v.Length() > limit {
		return v.Sub(v.Normalize().ScaleF(limit))
	}
	return Vector{0, 0}
}

// Round returns a new Vector with each component rounded to the nearest integer.
func (v Vector) Round() Vector {
	return V(math.Round(v.X), math.Round(v.Y))
}

// Floor returns a new Vector with each component rounded down to the nearest integer.
func (v Vector) Floor() Vector {
	return V(math.Floor(v.X), math.Floor(v.Y))
}

// Ceil returns a new Vector with each component rounded up to the nearest integer.
func (v Vector) Ceil() Vector {
	return V(math.Ceil(v.X), math.Ceil(v.Y))
}

// MoveInDirection moves the Vector in the direction of the angle by a given distance.
func (v Vector) MoveInDirection(angle, distance float64) Vector {
	return v.Add(V(math.Cos(angle), math.Sin(angle)).ScaleF(distance))
}

// Equals checks if two Vectors are equal within a small tolerance.
func (v Vector) Equals(other Vector) bool {
	return v.Sub(other).LengthSquared() < 0.00000000009
}

// Reflect reflects the vector against the given surface normal.
func (v Vector) Reflect(normal Vector) Vector {
	n := normal.Normalize()
	return v.Sub(n.ScaleF(2 * n.Dot(v)))
}

// Orthogonal returns a vector perpendicular to this vector (rotated 90 degrees counter-clockwise).
func (v Vector) Orthogonal() Vector {
	return V(-v.Y, v.X)
}

// Cross computes the 2D cross product (scalar) between this Vector and another.
func (v Vector) Cross(other Vector) float64 {
	return v.X*other.Y - v.Y*other.X
}
