package katsu2d

import (
	"reflect"
	"sync"

	"github.com/edwinsyarief/katsu2d/tween"
)

// --- ECS CORE ---

// nextComponentID is a counter to assign unique IDs to component types.
var (
	nextComponentID ComponentID
	// typeToID maps a component's reflect.Type to its unique ComponentID.
	typeToID = make(map[reflect.Type]ComponentID)
	// componentTypes maps a ComponentID back to its reflect.Type.
	componentTypes = make(map[ComponentID]reflect.Type)
	// mutex ensures thread safety for component registration.
	mutex sync.Mutex
)

// ComponentID is a unique identifier for a component type.
type ComponentID uint32

// RegisterComponent registers a component type and returns its unique ID.
// This should be called once for each component type at the beginning of the program.
func RegisterComponent[T any]() ComponentID {
	mutex.Lock()
	defer mutex.Unlock()
	t := reflect.TypeOf((*T)(nil)).Elem()
	if id, ok := typeToID[t]; ok {
		return id
	}
	id := nextComponentID
	nextComponentID++
	typeToID[t] = id
	componentTypes[id] = t
	return id
}

// Built-in component IDs. Registered once at init.
var (
	CTTransform ComponentID
	CTSprite    ComponentID
	CTAnimation ComponentID
	CTTween     ComponentID
	CTSequence  ComponentID
	CTText      ComponentID
)

func init() {
	CTTransform = RegisterComponent[Transform]()
	CTSprite = RegisterComponent[Sprite]()
	CTAnimation = RegisterComponent[Animation]()
	CTTween = RegisterComponent[tween.Tween]()
	CTSequence = RegisterComponent[tween.Sequence]()
	CTText = RegisterComponent[Text]()
}

// Entity is a unique identifier for an entity.
type Entity uint64

// World manages entities and their components.
type World struct {
	nextEntityID    Entity
	entities        map[Entity]struct{}
	componentStores map[reflect.Type]map[Entity]any
	entityMasks     map[Entity]uint64
	toRemove        []Entity
}

// NewWorld creates a new ECS world.
func NewWorld() *World {
	return &World{
		nextEntityID:    1,
		entities:        make(map[Entity]struct{}),
		componentStores: make(map[reflect.Type]map[Entity]any),
		entityMasks:     make(map[Entity]uint64),
		toRemove:        make([]Entity, 0),
	}
}

// CreateEntity creates a new entity.
func (self *World) CreateEntity() Entity {
	id := self.nextEntityID
	self.nextEntityID++
	self.entities[id] = struct{}{}
	self.entityMasks[id] = 0
	return id
}

// RemoveEntity marks an entity for removal (deferred until end of frame).
func (self *World) RemoveEntity(e Entity) {
	self.toRemove = append(self.toRemove, e)
}

// BatchRemoveEntities marks multiple entities for removal.
func (self *World) BatchRemoveEntities(entities ...Entity) {
	self.toRemove = append(self.toRemove, entities...)
}

// processRemovals removes marked entities.
// This should be called once per frame by the Update loop.
func (self *World) processRemovals() {
	// To handle potential duplicates, use a set
	removeSet := make(map[Entity]struct{})
	for _, e := range self.toRemove {
		removeSet[e] = struct{}{}
	}
	for e := range removeSet {
		for _, store := range self.componentStores {
			delete(store, e)
		}
		delete(self.entities, e)
		delete(self.entityMasks, e)
	}
	self.toRemove = self.toRemove[:0]
}

// AddComponent adds a component to an entity.
func (self *World) AddComponent(e Entity, comp any) {
	if _, ok := self.entities[e]; !ok {
		return // Entity does not exist
	}
	t := reflect.TypeOf(comp).Elem()
	id, ok := typeToID[t]
	if !ok {
		panic("Component not registered: " + t.Name())
	}
	if self.componentStores[t] == nil {
		self.componentStores[t] = make(map[Entity]any)
	}
	self.componentStores[t][e] = comp
	self.entityMasks[e] |= 1 << uint64(id)
}

// GetComponent retrieves a component from an entity using a ComponentID.
func (self *World) GetComponent(e Entity, id ComponentID) (any, bool) {
	t, ok := componentTypes[id]
	if !ok {
		return nil, false
	}
	store, ok := self.componentStores[t]
	if !ok {
		return nil, false
	}
	c, ok := store[e]
	return c, ok
}

// RemoveComponent removes a component from an entity.
func (self *World) RemoveComponent(e Entity, id ComponentID) {
	t, ok := componentTypes[id]
	if !ok {
		return
	}
	if store, ok := self.componentStores[t]; ok {
		delete(store, e)
	}
	self.entityMasks[e] &= ^(1 << uint64(id))
}

// Query returns entities that have all specified components.
func (self *World) Query(componentIDs ...ComponentID) []Entity {
	if len(componentIDs) == 0 {
		res := make([]Entity, 0, len(self.entities))
		for e := range self.entities {
			res = append(res, e)
		}
		return res
	}
	var mask uint64
	for _, id := range componentIDs {
		mask |= 1 << uint64(id)
	}
	res := make([]Entity, 0)
	for e := range self.entities {
		if self.entityMasks[e]&mask == mask {
			res = append(res, e)
		}
	}
	return res
}
