package katsu2d

import (
	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/hajimehoshi/ebiten/v2"
)

type Vertices []ebiten.Vertex
type Indices []uint16

type Point struct {
	X, Y float64
}

type Bound struct {
	Min Point
	Max Point
}

type EaseType int

const (
	Linear EaseType = iota
	QuadIn
	QuadOut
	QuadInOut
	QuadOutIn
	CubicIn
	CubicOut
	CubicInOut
	CubicOutIn
	QuartIn
	QuartOut
	QuartInOut
	QuartOutIn
	QuintIn
	QuintOut
	QuintInOut
	QuintOutIn
	SineIn
	SineOut
	SineInOut
	SineOutIn
	ExpoIn
	ExpoOut
	ExpoInOut
	ExpoOutIn
	CircIn
	CircOut
	CircInOut
	CircOutIn
	BackIn
	BackOut
	BackInOut
	BackOutIn
	BounceIn
	BounceOut
	BounceInOut
	BounceOutIn
	ElasticIn
	ElasticOut
	ElasticInOut
	ElasticOutIn
)

func EaseTypes[T ease.Float](et EaseType) ease.EaseFunc[T] {
	switch et {
	case QuadIn:
		return ease.QuadIn
	case QuadOut:
		return ease.QuadOut
	case QuadInOut:
		return ease.QuadInOut
	case QuadOutIn:
		return ease.QuadOutIn
	case CubicIn:
		return ease.CubicIn
	case CubicOut:
		return ease.CubicOut
	case CubicInOut:
		return ease.CubicInOut
	case CubicOutIn:
		return ease.CubicOutIn
	case QuartIn:
		return ease.QuartIn
	case QuartOut:
		return ease.QuartOut
	case QuartInOut:
		return ease.QuartInOut
	case QuartOutIn:
		return ease.QuartOutIn
	case QuintIn:
		return ease.QuintIn
	case QuintOut:
		return ease.QuintOut
	case QuintInOut:
		return ease.QuintInOut
	case QuintOutIn:
		return ease.QuintOutIn
	case SineIn:
		return ease.SineIn
	case SineOut:
		return ease.SineOut
	case SineInOut:
		return ease.SineInOut
	case SineOutIn:
		return ease.SineOutIn
	case ExpoIn:
		return ease.ExpoIn
	case ExpoOut:
		return ease.ExpoOut
	case ExpoInOut:
		return ease.ExpoInOut
	case ExpoOutIn:
		return ease.ExpoOutIn
	case CircIn:
		return ease.CircIn
	case CircOut:
		return ease.CircOut
	case CircInOut:
		return ease.CircInOut
	case CircOutIn:
		return ease.CircOutIn
	case BackIn:
		return ease.BackIn
	case BackOut:
		return ease.BackOut
	case BackInOut:
		return ease.BackInOut
	case BackOutIn:
		return ease.BackOutIn
	case BounceIn:
		return ease.BounceIn
	case BounceOut:
		return ease.BounceOut
	case BounceInOut:
		return ease.BounceInOut
	case BounceOutIn:
		return ease.BounceOutIn
	case ElasticIn:
		return ease.ElasticIn
	case ElasticOut:
		return ease.ElasticOut
	case ElasticInOut:
		return ease.ElasticInOut
	case ElasticOutIn:
		return ease.ElasticOutIn
	default:
		return ease.Linear
	}
}

// Bitmask is a type for bitmask operations.
type Bitmask uint

// Set adds the flag to the bitmask.
func (b *Bitmask) Set(flag Bitmask) {
	*b |= flag
}

// Unset removes the flag from the bitmask.
func (b *Bitmask) Unset(flag Bitmask) {
	*b &= ^flag
}

// Has checks if the bitmask has the flag set.
func (b Bitmask) Has(flag Bitmask) bool {
	return b&flag != 0
}

// Clear resets the bitmask to zero.
func (b *Bitmask) Clear() {
	*b = 0
}
