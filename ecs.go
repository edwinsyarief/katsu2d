package katsu2d

import (
	"reflect"
	"sync"
	"sync/atomic" // We will use atomic for the entity ID counter for better performance

	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
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
)

// ComponentID is a unique identifier for a component type.
type ComponentID uint32

// RegisterComponent registers a component type and returns its unique ID.
// This should be called once for each component type at the beginning of the program.
func RegisterComponent[T any]() ComponentID {
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
)

func init() {
	CTTransform = RegisterComponent[TransformComponent]()
	CTSprite = RegisterComponent[SpriteComponent]()
	CTAnimation = RegisterComponent[AnimationComponent]()
	CTTween = RegisterComponent[tween.Tween]()
	CTSequence = RegisterComponent[tween.Sequence]()
	CTFadeOverlay = RegisterComponent[overlays.FadeOverlay]()
	CTCinematicOverlay = RegisterComponent[overlays.CinematicOverlay]()
	CTText = RegisterComponent[TextComponent]()
	CTCooldown = RegisterComponent[managers.CooldownManager]()
	CTDelayer = RegisterComponent[managers.DelayManager]()
}

// Entity is a unique identifier for an entity.
type Entity uint64

// archetype is an internal structure to hold entities and their indices for fast removal.
type archetype struct {
	entities      []Entity
	entityIndices map[Entity]int
}

// World manages entities and their components.
type World struct {
	mu              sync.Mutex // Mutex to ensure thread safety for all state changes.
	nextEntityID    uint64     // Using uint64 to be compatible with atomic.
	entities        map[Entity]struct{}
	componentStores map[reflect.Type]map[Entity]any
	entityMasks     map[Entity]uint64
	archetypes      map[uint64]*archetype
	toRemove        []Entity
}

// NewWorld creates a new ECS world.
func NewWorld() *World {
	w := &World{
		nextEntityID:    1,
		entities:        make(map[Entity]struct{}),
		componentStores: make(map[reflect.Type]map[Entity]any),
		entityMasks:     make(map[Entity]uint64),
		archetypes:      make(map[uint64]*archetype),
		toRemove:        make([]Entity, 0),
	}
	// Initialize the default archetype for entities with no components.
	w.archetypes[0] = &archetype{
		entities:      make([]Entity, 0),
		entityIndices: make(map[Entity]int),
	}
	return w
}

// CreateEntity creates a new entity.
func (self *World) CreateEntity() Entity {
	self.mu.Lock()
	defer self.mu.Unlock()

	id := Entity(atomic.AddUint64(&self.nextEntityID, 1) - 1)
	self.entities[id] = struct{}{}
	self.entityMasks[id] = 0

	// Add the new entity to the archetype for mask 0
	arch := self.archetypes[0]
	arch.entityIndices[id] = len(arch.entities)
	arch.entities = append(arch.entities, id)
	return id
}

// RemoveEntity marks an entity for removal (deferred until end of frame).
func (self *World) RemoveEntity(e Entity) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.toRemove = append(self.toRemove, e)
}

// BatchRemoveEntities marks multiple entities for removal.
func (self *World) BatchRemoveEntities(entities ...Entity) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.toRemove = append(self.toRemove, entities...)
}

// processRemovals removes marked entities.
// This should be called once per frame by the Update loop.
func (self *World) processRemovals() {
	self.mu.Lock()
	defer self.mu.Unlock()

	// To handle potential duplicates, use a set
	removeSet := make(map[Entity]struct{})
	for _, e := range self.toRemove {
		removeSet[e] = struct{}{}
	}

	for e := range removeSet {
		// Clean up components and archetype
		self.removeEntityInternal(e)
	}

	self.toRemove = self.toRemove[:0]
}

// removeEntityInternal is a helper function to remove an entity from all stores.
// NOTE: This function assumes the mutex is already locked.
func (self *World) removeEntityInternal(e Entity) {
	// First, remove the entity from its archetype.
	mask := self.entityMasks[e]
	if arch, ok := self.archetypes[mask]; ok {
		if index, ok := arch.entityIndices[e]; ok {
			// Swap the entity to be removed with the last one in the slice
			lastIndex := len(arch.entities) - 1
			lastEntity := arch.entities[lastIndex]

			arch.entities[index] = lastEntity
			arch.entityIndices[lastEntity] = index

			// Truncate the slice and delete the index entry
			arch.entities = arch.entities[:lastIndex]
			delete(arch.entityIndices, e)

			// If the archetype is now empty, and it's not the base archetype (mask 0),
			// remove it from the map.
			if len(arch.entities) == 0 && mask != 0 {
				delete(self.archetypes, mask)
			}
		}
	}

	// Then, remove all of its components.
	for _, store := range self.componentStores {
		delete(store, e)
	}

	// Finally, remove the entity from the main entity and mask maps.
	delete(self.entities, e)
	delete(self.entityMasks, e)
}

// moveEntityArchetype moves an entity from old archetype to new.
// NOTE: This function assumes the mutex is already locked.
func (self *World) moveEntityArchetype(e Entity, oldMask, newMask uint64) {
	if oldMask == newMask {
		return
	}

	// Remove from the old archetype
	if oldArch, ok := self.archetypes[oldMask]; ok {
		if index, ok := oldArch.entityIndices[e]; ok {
			lastIndex := len(oldArch.entities) - 1
			lastEntity := oldArch.entities[lastIndex]

			oldArch.entities[index] = lastEntity
			oldArch.entityIndices[lastEntity] = index

			oldArch.entities = oldArch.entities[:lastIndex]
			delete(oldArch.entityIndices, e)

			// If the old archetype is now empty, and it's not the base archetype (mask 0),
			// remove it from the map.
			if len(oldArch.entities) == 0 && oldMask != 0 {
				delete(self.archetypes, oldMask)
			}
		}
	}

	// Add to the new archetype
	if _, ok := self.archetypes[newMask]; !ok {
		self.archetypes[newMask] = &archetype{
			entities:      make([]Entity, 0),
			entityIndices: make(map[Entity]int),
		}
	}
	newArch := self.archetypes[newMask]
	newArch.entityIndices[e] = len(newArch.entities)
	newArch.entities = append(newArch.entities, e)
}

// AddComponent adds a component to an entity.
func (self *World) AddComponent(e Entity, comp any) {
	self.mu.Lock()
	defer self.mu.Unlock()

	if _, ok := self.entities[e]; !ok {
		return // Entity does not exist
	}
	t := reflect.TypeOf(comp)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	id, ok := typeToID[t]
	if !ok {
		panic("Component not registered: " + t.Name())
	}
	if self.componentStores[t] == nil {
		self.componentStores[t] = make(map[Entity]any)
	}
	self.componentStores[t][e] = comp
	oldMask := self.entityMasks[e]
	self.entityMasks[e] |= 1 << uint64(id)
	newMask := self.entityMasks[e]
	self.moveEntityArchetype(e, oldMask, newMask)
}

// GetComponent retrieves a component from an entity using a ComponentID.
func (self *World) GetComponent(e Entity, id ComponentID) (any, bool) {
	self.mu.Lock()
	defer self.mu.Unlock()

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
	self.mu.Lock()
	defer self.mu.Unlock()

	t, ok := componentTypes[id]
	if !ok {
		return
	}
	if store, ok := self.componentStores[t]; ok {
		delete(store, e)
	}
	oldMask := self.entityMasks[e]
	self.entityMasks[e] &= ^(1 << uint64(id))
	newMask := self.entityMasks[e]
	self.moveEntityArchetype(e, oldMask, newMask)
}

// Query returns entities that have all specified components.
func (self *World) Query(componentIDs ...ComponentID) []Entity {
	self.mu.Lock()
	defer self.mu.Unlock()

	res := make([]Entity, 0)
	if len(componentIDs) == 0 {
		for _, arch := range self.archetypes {
			res = append(res, arch.entities...)
		}
		return res
	}
	var mask uint64
	for _, id := range componentIDs {
		mask |= 1 << uint64(id)
	}
	for archMask, arch := range self.archetypes {
		if archMask&mask == mask {
			res = append(res, arch.entities...)
		}
	}
	return res
}
