// package: katsu2d
// file: ecs.go

package katsu2d

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/edwinsyarief/katsu2d/tween"
)

// Entity is a unique identifier for game objects.
type Entity uint32

// ComponentID is a unique identifier for each component type.
type ComponentID uint32

var (
	// componentIDMap maps component types to their IDs.
	componentIDMap = make(map[reflect.Type]ComponentID)
	// nextComponentID is the next available component ID.
	nextComponentID ComponentID
	// Built-in component IDs.
	CTTransform        ComponentID
	CTSprite           ComponentID
	CTTween            ComponentID
	CTSequence         ComponentID
	CTAnimation        ComponentID
	CTFadeOverlay      ComponentID
	CTCinematicOverlay ComponentID
	CTCooldown         ComponentID
	CTDelayer          ComponentID
	CTText             ComponentID
)

func init() {
	// Register built-in components during initialization.
	// This ensures that our core components are always available.
	CTTransform = RegisterComponent[Transform]()
	CTSprite = RegisterComponent[Sprite]()
	CTTween = RegisterComponent[tween.Tween]()
	CTSequence = RegisterComponent[tween.Sequence]()
	CTAnimation = RegisterComponent[Animation]()
	CTFadeOverlay = RegisterComponent[overlays.FadeOverlay]()
	CTCinematicOverlay = RegisterComponent[overlays.CinematicOverlay]()
	CTCooldown = RegisterComponent[managers.CooldownManager]()
	CTDelayer = RegisterComponent[managers.DelayManager]()
	CTText = RegisterComponent[Text]()
}

// RegisterComponent registers a new component type and returns its ComponentID.
// This function should be called once per component type at program startup.
func RegisterComponent[T any]() ComponentID {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if id, ok := componentIDMap[t]; ok {
		return id // Component already registered, return its ID.
	}
	// Assign a new ID and increment the counter.
	id := nextComponentID
	nextComponentID++
	componentIDMap[t] = id
	return id
}

// entityLocation stores the archetype and index for an entity,
// allowing for fast lookups and state management.
type entityLocation struct {
	arch  *Archetype // The archetype the entity belongs to.
	index int        // The index of the entity within the archetype's dense arrays.
}

// Archetype groups entities with the exact same set of components.
// This is the core of a data-oriented ECS, enabling high-performance
// querying and cache efficiency.
type Archetype struct {
	id           string              // Unique key for the archetype (sorted component IDs).
	componentIDs []ComponentID       // The list of component IDs.
	componentMap map[ComponentID]int // A map for fast lookup of a component's index.
	entities     []Entity            // The dense array of entities.
	data         [][]any             // The dense arrays of component data.
	capacity     int                 // The current allocated capacity.
	length       int                 // The number of active entities.
}

// has checks if the archetype contains a specific component.
func (self *Archetype) has(ct ComponentID) bool {
	_, ok := self.componentMap[ct]
	return ok
}

// containsAll checks if the archetype contains all of the specified components.
func (self *Archetype) containsAll(cts []ComponentID) bool {
	for _, ct := range cts {
		if _, ok := self.componentMap[ct]; !ok {
			return false
		}
	}
	return true
}

// containsAny checks if the archetype has at least one of the specified components.
func (self *Archetype) containsAny(cts []ComponentID) bool {
	for _, ct := range cts {
		if _, ok := self.componentMap[ct]; ok {
			return true
		}
	}
	return false
}

// addEmptySlot adds an empty slot for a new entity. It handles growing the
// underlying arrays if the capacity is exceeded.
func (self *Archetype) addEmptySlot() int {
	if self.length == self.capacity {
		newCap := self.capacity * 2
		if newCap == 0 {
			newCap = 16 // Initial capacity
		}

		newEntities := make([]Entity, newCap)
		copy(newEntities, self.entities)
		self.entities = newEntities

		for i := range self.data {
			newData := make([]any, newCap)
			copy(newData, self.data[i])
			self.data[i] = newData
		}

		self.capacity = newCap
	}

	idx := self.length
	self.length++
	return idx
}

// swapRemove removes an entity at a given index by swapping it with the last
// entity in the archetype. This is a high-performance removal method for dense arrays.
func (self *Archetype) swapRemove(index int, world *World) {
	if self.length == 0 {
		return
	}

	last := self.length - 1

	// If the entity to remove is not the last one, we need to swap.
	if index != last {
		// Get the entity ID of the last element, which will be moved.
		swappedE := self.entities[last]

		// Copy the last entity's ID and component data to the target index.
		self.entities[index] = swappedE
		for i := range self.data {
			self.data[i][index] = self.data[i][last]
		}

		// IMPORTANT: Update the world's entityInfo for the entity that was swapped.
		// This is a critical step to ensure the entity still points to its correct location.
		world.entityInfo[swappedE] = entityLocation{arch: self, index: index}
	}

	// Now, clear the last slot and shrink the slice.
	// This prevents memory leaks and ensures no stale data remains.
	for i := range self.data {
		self.data[i][last] = nil
	}
	self.entities[last] = 0

	self.length--
}

// World manages all entities, archetypes, and systems.
type World struct {
	nextEntity Entity                    // Next available entity ID.
	archetypes map[string]*Archetype     // Map of archetype keys to archetypes.
	entityInfo map[Entity]entityLocation // Map of entity to its location.
	systems    []System                  // A slice to hold all registered systems.
}

// NewWorld creates a new ECS world with an empty archetype.
func NewWorld() *World {
	w := &World{
		archetypes: make(map[string]*Archetype),
		entityInfo: make(map[Entity]entityLocation),
		systems:    make([]System, 0),
	}
	// The empty archetype is the starting point for all new entities.
	w.getOrCreateArchetype([]ComponentID{})
	return w
}

// AddSystem adds a system to the world's list of systems.
func (self *World) AddSystem(s System) {
	self.systems = append(self.systems, s)
}

// UpdateSystems runs the update function for all registered systems.
func (self *World) UpdateSystems(dt float64) {
	for _, s := range self.systems {
		s.Update(self, dt)
	}
}

// getOrCreateArchetype retrieves or creates an archetype for a given set of components.
func (self *World) getOrCreateArchetype(ids []ComponentID) *Archetype {
	// We sort the IDs to create a consistent key for the archetype map.
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var key strings.Builder
	for i, id := range ids {
		if i > 0 {
			key.WriteByte(':')
		}
		fmt.Fprintf(&key, "%d", id)
	}
	k := key.String()

	if a, ok := self.archetypes[k]; ok {
		return a // Archetype already exists.
	}

	// Create a new archetype if one doesn't exist.
	initialCap := 16
	a := &Archetype{
		id:           k,
		componentIDs: append([]ComponentID{}, ids...),
		componentMap: make(map[ComponentID]int),
		entities:     make([]Entity, initialCap),
		data:         make([][]any, len(ids)),
		capacity:     initialCap,
		length:       0,
	}

	for i, id := range ids {
		a.componentMap[id] = i
		a.data[i] = make([]any, initialCap)
	}

	self.archetypes[k] = a
	return a
}

// NewEntity creates and returns a new unique entity ID.
func (self *World) NewEntity() Entity {
	e := self.nextEntity
	self.nextEntity++

	// New entities start in the empty archetype.
	empty := self.archetypes[""]
	idx := empty.addEmptySlot()
	empty.entities[idx] = e

	self.entityInfo[e] = entityLocation{arch: empty, index: idx}
	return e
}

// getComponentID retrieves the ID for a given component instance.
func getComponentID(c any) ComponentID {
	t := reflect.TypeOf(c).Elem()
	id, ok := componentIDMap[t]
	if !ok {
		panic("Unregistered component type; call RegisterComponent first")
	}
	return id
}

// AddComponent adds a component to an entity. This moves the entity
// to a new, more specific archetype.
func (self *World) AddComponent(e Entity, c any) {
	ct := getComponentID(c)
	loc, ok := self.entityInfo[e]
	if !ok {
		panic("Entity does not exist")
	}

	oldArch := loc.arch
	if oldArch.has(ct) {
		panic("Entity already has this component")
	}

	// Build the new archetype's component IDs.
	newIDs := append(oldArch.componentIDs, ct)
	newArch := self.getOrCreateArchetype(newIDs)
	newIndex := newArch.addEmptySlot()
	newArch.entities[newIndex] = e

	// Copy existing components from the old archetype to the new one.
	for _, oldID := range oldArch.componentIDs {
		oldIdx := oldArch.componentMap[oldID]
		oldComp := oldArch.data[oldIdx][loc.index]
		newIdx := newArch.componentMap[oldID]
		newArch.data[newIdx][newIndex] = oldComp
	}

	// Add the new component.
	newIdx := newArch.componentMap[ct]
	newArch.data[newIdx][newIndex] = c

	// Update the entity's location in the world.
	self.entityInfo[e] = entityLocation{arch: newArch, index: newIndex}

	// Remove the entity from its old archetype.
	oldArch.swapRemove(loc.index, self)
}

// RemoveComponents removes one or more components from an entity.
func (w *World) RemoveComponents(e Entity, cts ...ComponentID) {
	loc, ok := w.entityInfo[e]
	if !ok {
		return // Entity does not exist.
	}

	oldArch := loc.arch
	// Create a set of component IDs to be removed for fast lookups.
	removeSet := make(map[ComponentID]bool, len(cts))
	for _, ct := range cts {
		if oldArch.has(ct) {
			removeSet[ct] = true
		}
	}
	if len(removeSet) == 0 {
		return // No components to remove.
	}

	// Build the component IDs for the new archetype.
	var newIDs []ComponentID
	for _, id := range oldArch.componentIDs {
		if !removeSet[id] {
			newIDs = append(newIDs, id)
		}
	}

	// Get or create the new archetype.
	newArch := w.getOrCreateArchetype(newIDs)
	newIndex := newArch.addEmptySlot()
	newArch.entities[newIndex] = e

	// Copy components that are staying.
	for _, id := range newIDs {
		oldIdx := oldArch.componentMap[id]
		newIdx := newArch.componentMap[id]
		newArch.data[newIdx][newIndex] = oldArch.data[oldIdx][loc.index]
		oldArch.data[oldIdx][loc.index] = nil // Clear old reference to prevent memory leak.
	}

	// Update the entity's location.
	w.entityInfo[e] = entityLocation{arch: newArch, index: newIndex}

	// Remove the entity from its old archetype.
	oldArch.swapRemove(loc.index, w)
}

// GetComponent retrieves a component from an entity.
func (self *World) GetComponent(e Entity, ct ComponentID) any {
	loc, ok := self.entityInfo[e]
	if !ok {
		return nil
	}
	if !loc.arch.has(ct) {
		return nil
	}
	idx := loc.arch.componentMap[ct]
	return loc.arch.data[idx][loc.index]
}

// QueryAll returns all entities that have ALL of the specified components.
func (self *World) QueryAll(cts ...ComponentID) []Entity {
	// Initialize with a slice to ensure a valid empty slice is returned if no matches are found.
	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAll(cts) {
			entities = append(entities, a.entities[:a.length]...)
		}
	}
	return entities
}

// QueryAny returns all entities that have AT LEAST ONE of the specified components.
func (self *World) QueryAny(cts ...ComponentID) []Entity {
	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAny(cts) {
			entities = append(entities, a.entities[:a.length]...)
		}
	}
	return entities
}

// DestroyEntity removes an entity and all its components from the world.
func (w *World) DestroyEntity(e Entity) {
	loc, ok := w.entityInfo[e]
	if !ok {
		return // Entity does not exist.
	}
	arch := loc.arch
	arch.swapRemove(loc.index, w)
	delete(w.entityInfo, e)
}

// DestroyAllEntitiesWith destroys all entities that have all the specified components.
func (w *World) DestroyAllEntitiesWith(cts ...ComponentID) {
	for _, a := range w.archetypes {
		if a.containsAll(cts) {
			// Clear all entities in this archetype.
			for i := 0; i < a.length; i++ {
				e := a.entities[i]
				delete(w.entityInfo, e)
			}
			// Reset the length to effectively clear the archetype's data.
			a.length = 0
		}
	}
}

// DestroyAllEntities removes all entities from the world, resetting it to empty.
func (w *World) DestroyAllEntities() {
	for _, a := range w.archetypes {
		a.length = 0
	}
	w.entityInfo = make(map[Entity]entityLocation)
	w.nextEntity = 0
}
