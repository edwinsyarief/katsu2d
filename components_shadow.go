package katsu2d

import "image/color"

// ShadowType selects how a ShadowComponent renders a shadow or reflection.
type ShadowType int

const (
	// ShadowTypeProjected renders a directional skewed projection of the entity's sprite silhouette
	// (or a floor/water reflection when PreserveColor is true).
	ShadowTypeProjected ShadowType = iota
	// ShadowTypeDrop renders an offset silhouette copy of the entity's sprite.
	ShadowTypeDrop
	// ShadowTypeContact renders a soft anti-aliased elliptical contact blob under the caster
	// using a shared radial gradient texture.
	ShadowTypeContact
)

// ShadowComponent configures a cast shadow, drop shadow, contact blob, or reflection for an entity.
// It is a pure POD struct containing zero pointers and zero GPU image references.
type ShadowComponent struct {
	Type           ShadowType
	Angle          float64
	Length         float64
	Offset         Vector
	Scale          Vector
	RadiusX        float64
	RadiusY        float64
	Elevation      float64
	Color          color.RGBA
	Opacity        float64
	FadeOut        bool
	FadeAlpha      float64
	PreserveColor  bool
	UseGlobalLight bool
	Enabled        bool
}

// NewProjectedShadow creates a directional projected cast shadow with black tint,
// opacity 0.4, tip fade-out enabled, and global light inheritance enabled.
func NewProjectedShadow(angle, length float64) ShadowComponent {
	return ShadowComponent{
		Type:           ShadowTypeProjected,
		Angle:          angle,
		Length:         length,
		Scale:          Vector{X: 1, Y: 1},
		Color:          color.RGBA{R: 0, G: 0, B: 0, A: 255},
		Opacity:        0.4,
		FadeOut:        true,
		FadeAlpha:      0.0,
		UseGlobalLight: true,
		Enabled:        true,
	}
}

// NewDropShadow creates an offset silhouette drop shadow with black tint and opacity 0.4.
func NewDropShadow(offset Vector) ShadowComponent {
	return ShadowComponent{
		Type:    ShadowTypeDrop,
		Offset:  offset,
		Scale:   Vector{X: 1, Y: 1},
		Color:   color.RGBA{R: 0, G: 0, B: 0, A: 255},
		Opacity: 0.4,
		Enabled: true,
	}
}

// NewContactShadow creates a soft elliptical contact shadow with black tint and opacity 0.45.
func NewContactShadow(radiusX, radiusY float64) ShadowComponent {
	return ShadowComponent{
		Type:    ShadowTypeContact,
		Scale:   Vector{X: 1, Y: 1},
		RadiusX: radiusX,
		RadiusY: radiusY,
		Color:   color.RGBA{R: 0, G: 0, B: 0, A: 255},
		Opacity: 0.45,
		Enabled: true,
	}
}

// NewReflectionShadow creates a color-preserving vertical reflection anchored at the caster base.
func NewReflectionShadow(length float64) ShadowComponent {
	return ShadowComponent{
		Type:          ShadowTypeProjected,
		Angle:         0,
		Length:        length,
		Scale:         Vector{X: 1, Y: 1},
		Color:         color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Opacity:       0.35,
		FadeOut:       true,
		FadeAlpha:     0.0,
		PreserveColor: true,
		Enabled:       true,
	}
}

// ShadowLightComponent defines an optional global directional light source
// inherited by ShadowComponents with UseGlobalLight enabled.
type ShadowLightComponent struct {
	Angle   float64
	Length  float64
	Color   color.RGBA
	Opacity float64
	Enabled bool
}

// NewShadowLightComponent creates an enabled global directional shadow light
// with default black shadow tint.
func NewShadowLightComponent(angle, length, opacity float64) ShadowLightComponent {
	return ShadowLightComponent{
		Angle:   angle,
		Length:  length,
		Color:   color.RGBA{R: 0, G: 0, B: 0, A: 255},
		Opacity: opacity,
		Enabled: true,
	}
}
