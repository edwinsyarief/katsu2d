package katsu2d

import (
	"reflect"

	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
)

// ComponentID represents a unique identifier for a component type.
// Each component type in the game engine receives a unique ID during registration.
type ComponentID uint32

// Global component registry maps
var (
	nextComponentID ComponentID                          // Counter for generating unique component IDs
	typeToID        = make(map[reflect.Type]ComponentID) // Maps component types to their IDs
	componentTypes  = make(map[ComponentID]reflect.Type) // Maps component IDs to their types
)

// Built-in component IDs are registered during initialization.
// These constants provide quick access to commonly used component types.
var (
	CTTransform         ComponentID // Transform component for position, rotation, and scale
	CTSprite            ComponentID // Sprite component for rendering images
	CTAnimation         ComponentID // Animation component for sprite animations
	CTTween             ComponentID // Tween component for smooth value interpolation
	CTSequence          ComponentID // Sequence component for chaining animations
	CTFadeOverlay       ComponentID // Fade overlay for screen transitions
	CTCinematicOverlay  ComponentID // Cinematic overlay for cutscenes
	CTText              ComponentID // Text rendering component
	CTCooldown          ComponentID // Cooldown management for timed actions
	CTDelayer           ComponentID // Delay manager for delayed actions
	CTParticleEmitter   ComponentID // Particle system emitter
	CTParticle          ComponentID // Individual particle component
	CTTag               ComponentID // Tag component for entity identification
	CTTileMap           ComponentID // Tilemap component for level layouts
	CTInput             ComponentID // Input handling component
	CTBasicCamera       ComponentID // Simple camera component
	CTCamera            ComponentID // Advanced camera component
	CTGrass             ComponentID // Individual grass element component
	CTGrassController   ComponentID // Controls grass field behavior
	CTParent            ComponentID // Parent component for scene graph hierarchy
	CTOrderable         ComponentID // Orderable component for render sorting
	CTFoliage           ComponentID // Foliage component for vegetation
	CTFoliageController ComponentID // Foliage controller for managing foliage physics
)

// RegisterComponent registers a component type and returns its unique ID.
// This generic function should be called once for each component type during initialization.
// Type parameter T represents the component type to register.
// Returns: A unique ComponentID for the registered type.
// If the component type is already registered, returns its existing ID.
func RegisterComponent[T any]() ComponentID {
	var t T
	compType := reflect.TypeOf(t)
	if id, ok := typeToID[compType]; ok {
		return id
	}
	id := nextComponentID
	nextComponentID++
	typeToID[compType] = id
	componentTypes[id] = compType
	return id
}

// init registers all built-in component types at program startup.
// This ensures all core components have valid IDs before any game systems are initialized.
func init() {
	CTTransform = RegisterComponent[*TransformComponent]()
	CTSprite = RegisterComponent[*SpriteComponent]()
	CTAnimation = RegisterComponent[*AnimationComponent]()
	CTTween = RegisterComponent[*tween.Tween]()
	CTSequence = RegisterComponent[*tween.Sequence]()
	CTFadeOverlay = RegisterComponent[*FadeOverlayComponent]()
	CTCinematicOverlay = RegisterComponent[*CinematicOverlayComponent]()
	CTText = RegisterComponent[*TextComponent]()
	CTCooldown = RegisterComponent[*managers.CooldownManager]()
	CTDelayer = RegisterComponent[*managers.DelayManager]()
	CTParticleEmitter = RegisterComponent[*ParticleEmitterComponent]()
	CTParticle = RegisterComponent[*ParticleComponent]()
	CTTag = RegisterComponent[*TagComponent]()
	CTInput = RegisterComponent[*InputComponent]()
	CTBasicCamera = RegisterComponent[*BasicCameraComponent]()
	CTCamera = RegisterComponent[*CameraComponent]()
	CTGrass = RegisterComponent[*GrassComponent]()
	CTGrassController = RegisterComponent[*GrassControllerComponent]()
	CTParent = RegisterComponent[*ParentComponent]()
	CTOrderable = RegisterComponent[*OrderableComponent]()
	CTFoliage = RegisterComponent[*FoliageComponent]()
	CTFoliageController = RegisterComponent[*FoliageControllerComponent]()
}
