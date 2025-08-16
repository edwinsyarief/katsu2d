package katsu2d

import (
	"image"
	"image/color"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/ease"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
)

const (
	// SpotlightOvershootFactor is the additional scaling factor for the spotlight effect.
	SpotlightOvershootFactor = 0.25
)

// TagComponent is a component that provides a simple string tag for an entity.
// This can be used for querying and grouping.
type TagComponent struct {
	Tag string
}

// Make a new instance of TagComponent.
func NewTagComponent(tag string) *TagComponent {
	return &TagComponent{Tag: tag}
}

// TransformComponent component defines position, offset, origin, scale, rotation, and z-index.
type TransformComponent struct {
	*ebimath.Transform
	Z float64 // for draw order
}

// NewTransformComponent creates a new Transform component with default values.
func NewTransformComponent() *TransformComponent {
	return &TransformComponent{
		Transform: ebimath.T(),
	}
}

// SetZ safely updates the Z value of a transform.
// If the Z value changes, it marks the world as "dirty"
// to signal that a sort is needed.
func (self *TransformComponent) SetZ(world *World, z float64) {
	if self.Z != z {
		self.Z = z
		// Signal the world that the Z-order is no longer valid.
		world.MarkZDirty()
	}
}

// SpriteComponent component defines texture, source rect, destination size, and color tint.
type SpriteComponent struct {
	TextureID int
	SrcX      float32
	SrcY      float32
	SrcW      float32 // if 0, use whole texture width
	SrcH      float32 // if 0, use whole texture height
	DstW      float32 // if 0, use SrcW
	DstH      float32 // if 0, use SrcH
	Color     color.RGBA
	Opacity   float32
}

// NewSpriteComponent creates a new Sprite component for a given texture and destination size.
func NewSpriteComponent(textureID, width, height int) *SpriteComponent {
	return &SpriteComponent{
		TextureID: textureID,
		DstW:      float32(width),
		DstH:      float32(height),
		Color:     color.RGBA{255, 255, 255, 255},
		Opacity:   1.0,
	}
}

// GetSourceRect calculates the source rectangle coordinates and size.
func (self *SpriteComponent) GetSourceRect(textureWidth, textureHeight float32) (x, y, w, h float32) {
	x, y = self.SrcX, self.SrcY
	w, h = self.SrcW, self.SrcH
	if w == 0 || h == 0 {
		w, h = textureWidth, textureHeight
	}
	return
}

// GetDestSize calculates the destination size, using source size as a fallback.
func (self *SpriteComponent) GetDestSize(sourceWidth, sourceHeight float32) (w, h float32) {
	w, h = self.DstW, self.DstH
	if w == 0 {
		w = sourceWidth
	}
	if h == 0 {
		h = sourceHeight
	}
	return
}

// AnimMode defines animation playback modes.
type AnimMode int

const (
	AnimLoop      AnimMode = iota // Loop forever
	AnimOnce                      // Play once and stop
	AnimBoomerang                 // Forward then backward loop
)

// AnimationComponent component for sprite frame animations.
type AnimationComponent struct {
	Frames    []image.Rectangle
	Speed     float64  // Seconds per frame
	Elapsed   float64  // Time since last frame
	Current   int      // Current frame index
	Mode      AnimMode // Animation mode
	Direction bool     // For boomerang: true forward, false backward
	Active    bool     // Is animation playing
}

// TextAlignment defines text alignment modes.
type TextAlignment int

const (
	TextAlignmentBottomRight TextAlignment = iota
	TextAlignmentMiddleRight
	TextAlignmentTopRight

	TextAlignmentBottomCenter
	TextAlignmentMiddleCenter
	TextAlignmentTopCenter

	TextAlignmentBottomLeft
	TextAlignmentMiddleLeft
	TextAlignmentTopLeft
)

// alignmentOffsets stores pre-calculated offset functions.
// Each function takes the text's width (w) and height (h) and returns
// the (x, y) coordinates of the top-left corner of the text's bounding box.
// These coordinates are relative to the desired alignment point.
var alignmentOffsets = map[TextAlignment]func(w, h float64) (float64, float64){
	// Right-aligned offsets: the x-coordinate of the bounding box is a negative offset from the alignment point.
	// Y-offsets are calculated from the bottom of the bounding box to the alignment point.
	TextAlignmentTopRight:    func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentMiddleRight: func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomRight: func(w, h float64) (float64, float64) { return 0, -h },

	// Center-aligned offsets: the x-coordinate of the bounding box is offset by half its width.
	// Y-offsets are calculated from the top, middle, or bottom of the bounding box.
	TextAlignmentTopCenter:    func(w, h float64) (float64, float64) { return -w / 2, 0 },
	TextAlignmentMiddleCenter: func(w, h float64) (float64, float64) { return -w / 2, -h / 2 },
	TextAlignmentBottomCenter: func(w, h float64) (float64, float64) { return -w / 2, -h },

	// Left-aligned offsets: the x-coordinate of the bounding box is the same as the alignment point.
	// Y-offsets are calculated from the top, middle, or bottom of the bounding box.
	TextAlignmentTopLeft:    func(w, h float64) (float64, float64) { return 0, 0 },
	TextAlignmentMiddleLeft: func(w, h float64) (float64, float64) { return 0, -h / 2 },
	TextAlignmentBottomLeft: func(w, h float64) (float64, float64) { return 0, -h },
}

// TextComponent component for drawing text.
type TextComponent struct {
	Alignment         TextAlignment
	Caption           string
	Size, lineSpacing float64
	Color             color.RGBA

	fontFace *text.GoTextFace

	// Cache for text measurements
	cachedWidth, cachedHeight float64
	cachedText                string
}

// NewTextComponent creates a new Text component with the specified font source, caption, size, and color.
func NewTextComponent(source *text.GoTextFaceSource, caption string, size float64, col color.RGBA) *TextComponent {
	fontFace := &text.GoTextFace{
		Source:    source,
		Direction: text.DirectionLeftToRight,
		Size:      size,
		Language:  language.English,
	}
	result := &TextComponent{
		Caption:  caption,
		Size:     size,
		Color:    col,
		fontFace: fontFace,
	}
	result.updateCache()
	return result
}

// updateCache updates the cached measurements for the text if the caption has changed.
func (self *TextComponent) updateCache() {
	if self.cachedText != self.Caption {
		self.cachedWidth, self.cachedHeight = text.Measure(self.Caption, self.fontFace, self.lineSpacing)
		self.cachedText = self.Caption
	}
}

func (self *TextComponent) LineSpacing() float64 {
	return self.lineSpacing
}

// SetLineSpacing sets the line spacing for the text and returns the text for chaining.
func (self *TextComponent) SetLineSpacing(spacing float64) *TextComponent {
	self.lineSpacing = spacing
	self.updateCache()
	return self
}

// SetAlignment sets the alignment for the text and returns the Text for chaining.
func (self *TextComponent) SetAlignment(alignment TextAlignment) *TextComponent {
	self.Alignment = alignment
	return self
}

func (self *TextComponent) GetOffset() (float64, float64) {
	if offsetFunc, ok := alignmentOffsets[self.Alignment]; ok {
		offsetX, offsetY := offsetFunc(self.cachedWidth, self.cachedHeight)
		return offsetX, offsetY
	}

	return 0, 0
}

// SetText sets the caption for the text and returns the Text for chaining.
func (self *TextComponent) SetText(text string) *TextComponent {
	if self.Caption != text {
		self.Caption = text
		self.updateCache()
	}
	return self
}

// SetFontFace sets the font face for the text and returns the Text for chaining.
func (self *TextComponent) SetFontFace(fontFace *text.GoTextFace) *TextComponent {
	self.fontFace = fontFace
	self.updateCache()
	return self
}

func (self *TextComponent) FontFace() *text.GoTextFace {
	return self.fontFace
}

// SetSize sets the size for the text and returns the Text for chaining.
func (self *TextComponent) SetSize(size float64) *TextComponent {
	self.fontFace.Size = size
	self.updateCache()
	return self
}

// SetColor sets the color for the text and returns the Text for chaining.
func (self *TextComponent) SetColor(color color.RGBA) *TextComponent {
	self.Color = color
	return self
}

// SetOpacity sets the opacity by adjusting the alpha channel of the color and returns the Text for chaining.
func (self *TextComponent) SetOpacity(opacity float64) *TextComponent {
	val := ebimath.Min(ebimath.Max(opacity, 0.0), 1.0)

	col := self.Color
	col.A = uint8(255 * val)
	self.SetColor(col)

	return self
}

// ParticleComponent holds the dynamic state of a single particle.
type ParticleComponent struct {
	Gravity, Velocity ebimath.Vector
	Lifetime          float64
	TotalLifetime     float64
	InitialColor      color.RGBA
	TargetColor       color.RGBA
	InitialScale      float64
	TargetScale       float64
	InitialRotation   float64
	TargetRotation    float64
}

// ParticleEmitterComponent holds the configuration for a particle effect.
type ParticleEmitterComponent struct {
	Active                                           bool // If true, continuously spawns particles
	BurstCount                                       int  // If > 0, spawns this many particles in one burst, then resets to 0
	EmitRate                                         float64
	MaxParticles                                     int
	ParticleLifetime                                 float64
	ParticleSpawnOffset                              ebimath.Vector // Random offset from emitter position
	InitialParticleSpeedMin, InitialParticleSpeedMax float64
	InitialColorMin, InitialColorMax                 color.RGBA
	TargetColorMin, TargetColorMax                   color.RGBA
	FadeOut                                          bool
	Gravity                                          ebimath.Vector
	TextureIDs                                       []int
	BlendMode                                        ebiten.Blend
	MinScale, MaxScale                               float64 // Initial scale range
	TargetScaleMin, TargetScaleMax                   float64 // Target scale range at end of life
	MinRotation, MaxRotation                         float64 // Initial rotation range
	EndRotationMin, EndRotationMax                   float64 // Target rotation range at end of life

	// Internal state
	lastEmitTime time.Time
	spawnCounter float64
}

// NewParticleEmitterComponent creates and returns a new ParticleEmitterComponent with a texture ID.
func NewParticleEmitterComponent(textureIDs []int) *ParticleEmitterComponent {
	return &ParticleEmitterComponent{
		TextureIDs:      textureIDs,
		lastEmitTime:    time.Now(),
		InitialColorMin: color.RGBA{255, 255, 255, 255},
		InitialColorMax: color.RGBA{255, 255, 255, 255},
		TargetColorMin:  color.RGBA{255, 255, 255, 255},
		TargetColorMax:  color.RGBA{255, 255, 255, 255},
		BlendMode:       ebiten.BlendSourceOver,
		MinScale:        1.0,
		MaxScale:        1.0,
		TargetScaleMin:  1.0,
		TargetScaleMax:  1.0,
		MinRotation:     0,
		MaxRotation:     0,
		EndRotationMin:  0,
		EndRotationMax:  0,
	}
}

type FadeType int

const (
	FadeTypeOut FadeType = iota
	FadeTypeIn
)

type FadeOverlayComponent struct {
	FadeType    FadeType
	FadeColor   color.RGBA
	Overlay     *ebiten.Image
	Tween       *tween.Tween
	CurrentFade float32
	Finished    bool
	Callback    func()
}

func NewFadeOverlayComponent(fadeType FadeType, fadeColor color.RGBA, duration float64, callback func()) *FadeOverlayComponent {
	begin, end := float32(0.0), float32(1.0)
	easing := ease.CubicInOut

	if fadeType == FadeTypeIn {
		begin, end = 1.0, 0.0
		easing = ease.QuadInOut
	}

	overlay := ebiten.NewImage(1, 1)
	overlay.Fill(fadeColor)

	return &FadeOverlayComponent{
		FadeType:  fadeType,
		FadeColor: fadeColor,
		Overlay:   overlay,
		Tween:     tween.New(begin, end, float32(duration), easing),
		Callback:  callback,
	}
}

func (self *FadeOverlayComponent) SetDelay(delay float64) {
	self.Tween.SetDelay(float32(delay))
}

// CinematicType defines the type of cinematic effect.
type CinematicType int

const (
	CinematicMovie     CinematicType = iota // Letterbox-style effect
	CinematicSpotlight                      // Circular spotlight effect
)

// CinematicOverlayType defines the transition direction.
type CinematicOverlayType int

const (
	CinematicTransitionIn  CinematicOverlayType = iota // Fade in
	CinematicTransitionOut                             // Fade out
)

// CinematicOverlayComponent manages a visual overlay for cinematic effects in a game.
type CinematicOverlayComponent struct {
	CinematicType                                          CinematicType
	StartType, EndType                                     CinematicOverlayType
	StartFade, EndFade                                     bool
	AutoFinish, CinematicFinished, DelayFinished, Finished bool
	Radius, StartSpeed, EndSpeed, CinematicDelay           float64
	CinematicAlphaValue                                    float64
	TransitionValue                                        float64
	Width, Height                                          int
	SpotlightMaxValue                                      float64 // Precomputed max value for spotlight

	Offset               ebimath.Vector
	OverlayColor         color.RGBA
	OverlayOpacity       float64
	Overlay, Placeholder *ebiten.Image
	RenderTarget         *ebiten.Image // Renamed from temp for clarity

	Tween   *tween.Sequence
	Delayer *managers.DelayManager

	lastStepOnce sync.Once

	Callback func()
}

// NewCinematicOverlayComponent creates a new cinematic overlay with the specified parameters.
func NewCinematicOverlayComponent(width, height int, col color.RGBA, opacity float64, cinematicType CinematicType,
	startType, endType CinematicOverlayType, startFade, endFade, autoFinish bool,
	cinematicDelay, radius, startSpeed, endSpeed float64, offset ebimath.Vector, callback func()) *CinematicOverlayComponent {
	result := &CinematicOverlayComponent{
		OverlayColor:   col,
		OverlayOpacity: opacity,
		Width:          width,
		Height:         height,
		CinematicType:  cinematicType,
		StartType:      startType,
		EndType:        endType,
		StartFade:      startFade,
		EndFade:        endFade,
		AutoFinish:     autoFinish,
		CinematicDelay: cinematicDelay,
		Radius:         radius,
		StartSpeed:     startSpeed,
		EndSpeed:       endSpeed,
		Offset:         offset,
		Delayer:        managers.NewDelayManager(),
		Callback:       callback,
	}

	// Configure delayer for cinematic delay
	result.Delayer.Add("cinematic_delay", cinematicDelay, func() {
		result.DelayFinished = true
	})

	// Initialize image buffers
	result.RenderTarget = ebiten.NewImage(width, height)
	result.Overlay = ebiten.NewImage(1, 1)
	result.Placeholder = ebiten.NewImage(1, 1)
	result.Overlay.Fill(result.OverlayColor)
	result.Placeholder.Fill(color.RGBA{R: 255, G: 255, B: 255, A: 255})
	result.CinematicAlphaValue = 1.0

	// Precompute spotlight max value if applicable
	if result.CinematicType == CinematicSpotlight {
		result.SpotlightMaxValue = float64(height) * (1 + SpotlightOvershootFactor)
	}

	// Initialize the starting tween
	result.Tween = result.createStartTween()

	return result
}

// createStartTween generates the initial tween sequence based on cinematic type and start type.
func (self *CinematicOverlayComponent) createStartTween() *tween.Sequence {
	if self.CinematicType == CinematicMovie {
		if self.StartType == CinematicTransitionIn {
			self.TransitionValue = 0.0
			maxValue := self.Radius / float64(self.Height)
			return tween.NewSequence(
				tween.New(0.0, float32(maxValue), float32(self.StartSpeed), ease.QuadInOut),
			)
		}
		self.TransitionValue = 1.0
		return tween.NewSequence(
			tween.New(1.0, float32(self.Radius/float64(self.Height)), float32(self.StartSpeed), ease.QuadInOut),
		)
	}

	// Spotlight type
	if self.StartType == CinematicTransitionIn {
		self.TransitionValue = 0.0
		return tween.NewSequence(
			tween.New(0.0, float32(self.Radius), float32(self.StartSpeed), ease.QuadInOut),
		)
	}
	self.TransitionValue = self.SpotlightMaxValue
	return tween.NewSequence(
		tween.New(float32(self.SpotlightMaxValue), float32(self.Radius), float32(self.StartSpeed), ease.QuadInOut),
	)
}

// createEndTween generates the tween for the end phase.
func (self *CinematicOverlayComponent) createEndTween() *tween.Tween {
	if self.CinematicType == CinematicMovie {
		if self.EndType == CinematicTransitionIn {
			return tween.New(float32(self.Radius)/float32(self.Height), 1.0, float32(self.EndSpeed), ease.QuadIn)
		}
		return tween.New(float32(self.Radius)/float32(self.Height), 0.0, float32(self.EndSpeed), ease.QuadOut)
	}
	// Spotlight type
	if self.EndType == CinematicTransitionIn {
		return tween.New(float32(self.Radius), float32(self.SpotlightMaxValue), float32(self.EndSpeed), ease.BackIn)
	}
	return tween.New(float32(self.Radius), 0.0, float32(self.EndSpeed), ease.BackOut)
}

// setupLastStep configures the end phase tween.
func (self *CinematicOverlayComponent) setupLastStep() {
	self.Tween.Remove(0)
	self.Tween.Add(self.createEndTween())
	self.Tween.Reset()
}

// EndCinematic manually triggers the end phase if not auto-finished.
func (self *CinematicOverlayComponent) EndCinematic() {
	if self.AutoFinish || !self.CinematicFinished {
		return
	}
	self.Delayer.Activate("cinematic_delay")
}
