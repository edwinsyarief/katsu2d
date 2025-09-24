package katsu2d

import (
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/lazyecs"
)

// Built-in component IDs are registered during initialization.
// These constants provide quick access to commonly used component types.
var (
	CTTransform         lazyecs.ComponentID // Transform component for position, rotation, and scale
	CTSprite            lazyecs.ComponentID // Sprite component for rendering images
	CTAnimation         lazyecs.ComponentID // Animation component for sprite animations
	CTTween             lazyecs.ComponentID // Tween component for smooth value interpolation
	CTSequence          lazyecs.ComponentID // Sequence component for chaining animations
	CTFadeOverlay       lazyecs.ComponentID // Fade overlay for screen transitions
	CTCinematicOverlay  lazyecs.ComponentID // Cinematic overlay for cutscenes
	CTText              lazyecs.ComponentID // Text rendering component
	CTCooldown          lazyecs.ComponentID // Cooldown management for timed actions
	CTDelayer           lazyecs.ComponentID // Delay manager for delayed actions
	CTParticleEmitter   lazyecs.ComponentID // Particle system emitter
	CTParticle          lazyecs.ComponentID // Individual particle component
	CTTag               lazyecs.ComponentID // Tag component for entity identification
	CTTileMap           lazyecs.ComponentID // Tilemap component for level layouts
	CTInput             lazyecs.ComponentID // Input handling component
	CTBasicCamera       lazyecs.ComponentID // Simple camera component
	CTCamera            lazyecs.ComponentID // Advanced camera component
	CTGrass             lazyecs.ComponentID // Individual grass element component
	CTGrassController   lazyecs.ComponentID // Controls grass field behavior
	CTParent            lazyecs.ComponentID // Parent component for scene graph hierarchy
	CTOrderable         lazyecs.ComponentID // Orderable component for render sorting
	CTFoliage           lazyecs.ComponentID // Foliage component for vegetation
	CTFoliageController lazyecs.ComponentID // Foliage controller for managing foliage physics
	CTShape             lazyecs.ComponentID // Shape component
)

// init registers all built-in component types at program startup.
// This ensures all core components have valid IDs before any game systems are initialized.
func init() {
	CTTransform = lazyecs.RegisterComponent[TransformComponent]()
	CTSprite = lazyecs.RegisterComponent[SpriteComponent]()
	CTAnimation = lazyecs.RegisterComponent[AnimationComponent]()
	CTTween = lazyecs.RegisterComponent[tween.Tween]()
	CTSequence = lazyecs.RegisterComponent[tween.Sequence]()
	CTFadeOverlay = lazyecs.RegisterComponent[FadeOverlayComponent]()
	CTCinematicOverlay = lazyecs.RegisterComponent[CinematicOverlayComponent]()
	CTText = lazyecs.RegisterComponent[TextComponent]()
	CTCooldown = lazyecs.RegisterComponent[managers.CooldownManager]()
	CTDelayer = lazyecs.RegisterComponent[managers.DelayManager]()
	CTParticleEmitter = lazyecs.RegisterComponent[ParticleEmitterComponent]()
	CTParticle = lazyecs.RegisterComponent[ParticleComponent]()
	CTTag = lazyecs.RegisterComponent[TagComponent]()
	CTInput = lazyecs.RegisterComponent[InputComponent]()
	CTBasicCamera = lazyecs.RegisterComponent[BasicCameraComponent]()
	CTCamera = lazyecs.RegisterComponent[CameraComponent]()
	CTGrass = lazyecs.RegisterComponent[GrassComponent]()
	CTGrassController = lazyecs.RegisterComponent[GrassControllerComponent]()
	CTParent = lazyecs.RegisterComponent[ParentComponent]()
	CTOrderable = lazyecs.RegisterComponent[OrderableComponent]()
	CTFoliage = lazyecs.RegisterComponent[FoliageComponent]()
	CTFoliageController = lazyecs.RegisterComponent[FoliageControllerComponent]()
	CTShape = lazyecs.RegisterComponent[ShapeComponent]()
}
