package katsu2d

import (
	"reflect"

	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/tween"
)

// ComponentID is a unique identifier for a component type.
type ComponentID uint32

// Global component registry
var (
	nextComponentID ComponentID
	typeToID        = make(map[reflect.Type]ComponentID)
	componentTypes  = make(map[ComponentID]reflect.Type)
)

// Built-in component IDs. Registered once at init.
var (
	CTTransform        ComponentID
	CTSprite           ComponentID
	CTAnimation        ComponentID
	CTTween            ComponentID
	CTSequence         ComponentID
	CTFadeOverlay      ComponentID
	CTCinematicOverlay ComponentID
	CTText             ComponentID
	CTCooldown         ComponentID
	CTDelayer          ComponentID
	CTParticleEmitter  ComponentID
	CTParticle         ComponentID
	CTTag              ComponentID
	CTTileMap          ComponentID
	CTInput            ComponentID
	CTBasicCamera      ComponentID
	CTCamera           ComponentID
	CTGrass            ComponentID
	CTGrassController  ComponentID
)

// RegisterComponent registers a component type and returns its unique ID.
// This should be called once for each component type at the beginning of the program.
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
}
