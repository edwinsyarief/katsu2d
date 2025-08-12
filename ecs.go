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
// Call this once per custom component type before adding any instances.
// Example: var CTMyComponent = game2d.RegisterComponent[*MyComponent]()
func RegisterComponent[T any]() ComponentID {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if id, ok := componentIDMap[t]; ok {
		return id // Already registered
	}
	id := nextComponentID
	nextComponentID++
	componentIDMap[t] = id
	return id
}

// Archetype groups entities with the exact same set of components for efficient storage and querying.
type Archetype struct {
	id           string              // Unique key for the archetype (sorted component IDs as string)
	componentIDs []ComponentID       // List of component IDs in this archetype
	componentMap map[ComponentID]int // Map from component ID to data slice index
	entities     []Entity            // List of entities in this archetype
	data         [][]any             // Dense arrays of component data, one per component type
	capacity     int                 // Current capacity of the arrays
	length       int                 // Number of active entities
}

// has checks if the archetype includes a specific component.
func (self *Archetype) has(ct ComponentID) bool {
	_, ok := self.componentMap[ct]
	return ok
}

// containsAll checks if the archetype has all specified components.
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

// addEmptySlot adds a new slot for an entity and returns its index, growing arrays if needed.
func (self *Archetype) addEmptySlot() int {
	if self.length == self.capacity {
		newCap := self.capacity * 2
		if newCap == 0 {
			newCap = 16
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

// swapRemove removes an entity at the given index by swapping with the last and shrinking.
func (self *Archetype) swapRemove(index int) {
	if self.length == 0 {
		return
	}
	last := self.length - 1
	if index == last {
		self.length--
		return
	}
	for i := range self.data {
		self.data[i][index] = self.data[i][last]
		self.data[i][last] = nil
	}
	swappedE := self.entities[last]
	self.entities[index] = swappedE
	// Clear the old last position to avoid a "zombie" reference.
	self.entities[last] = 0
	self.length--
}

// entityLocation stores where an entity is located (archetype and index).
type entityLocation struct {
	arch  *Archetype
	index int
}

// World manages all entities, archetypes, and components.
type World struct {
	nextEntity Entity                    // Next available entity ID
	archetypes map[string]*Archetype     // Map of archetype keys to archetypes
	entityInfo map[Entity]entityLocation // Map of entity to its location
}

// NewWorld creates a new ECS world with an empty archetype.
func NewWorld() *World {
	w := &World{
		archetypes: make(map[string]*Archetype),
		entityInfo: make(map[Entity]entityLocation),
	}
	// Create empty archetype
	w.getOrCreateArchetype([]ComponentID{})

	return w
}

// getOrCreateArchetype retrieves or creates an archetype for the given component IDs.
func (self *World) getOrCreateArchetype(ids []ComponentID) *Archetype {
	// Sort IDs for consistent key
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
		return a
	}
	// Create new archetype
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

// NewEntity creates a new entity in the empty archetype.
func (self *World) NewEntity() Entity {
	e := self.nextEntity
	self.nextEntity++
	empty := self.archetypes[""]
	idx := empty.addEmptySlot()
	empty.entities[idx] = e
	self.entityInfo[e] = entityLocation{arch: empty, index: idx}
	return e
}

// getComponentID retrieves the ID for a component instance.
func getComponentID(c any) ComponentID {
	t := reflect.TypeOf(c).Elem()
	id, ok := componentIDMap[t]
	if !ok {
		panic("Unregistered component type; call RegisterComponent first")
	}
	return id
}

// AddComponent adds a component to an entity, moving it to a new archetype if needed.
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
	// Create new archetype with added component
	newIDs := append(oldArch.componentIDs, ct)
	newArch := self.getOrCreateArchetype(newIDs)
	newIndex := newArch.addEmptySlot()
	newArch.entities[newIndex] = e
	// Copy existing components
	for _, oldID := range oldArch.componentIDs {
		oldIdx := oldArch.componentMap[oldID]
		oldComp := oldArch.data[oldIdx][loc.index]
		newIdx := newArch.componentMap[oldID]
		newArch.data[newIdx][newIndex] = oldComp
	}
	// Add new component
	newIdx := newArch.componentMap[ct]
	newArch.data[newIdx][newIndex] = c
	// Update entity location
	self.entityInfo[e] = entityLocation{arch: newArch, index: newIndex}
	// Remove from old archetype
	oldArch.swapRemove(loc.index)
	// Update swapped entity's location if necessary
	if oldArch.length > loc.index {
		swappedE := oldArch.entities[loc.index]
		self.entityInfo[swappedE] = entityLocation{arch: oldArch, index: loc.index}
	}
}

// RemoveComponents removes the specified components from the entity, if present.
func (w *World) RemoveComponents(e Entity, cts ...ComponentID) {
	loc, ok := w.entityInfo[e]
	if !ok {
		return // Entity does not exist
	}
	oldArch := loc.arch
	// Create set of components to remove for fast lookup
	removeSet := make(map[ComponentID]bool, len(cts))
	for _, ct := range cts {
		if oldArch.has(ct) {
			removeSet[ct] = true
		}
	}
	if len(removeSet) == 0 {
		return // No components to remove
	}
	// Build new component IDs without the removed ones
	var newIDs []ComponentID
	for _, id := range oldArch.componentIDs {
		if !removeSet[id] {
			newIDs = append(newIDs, id)
		}
	}
	// Get or create new archetype
	newArch := w.getOrCreateArchetype(newIDs)
	newIndex := newArch.addEmptySlot()
	newArch.entities[newIndex] = e
	// Copy remaining components
	for _, id := range newIDs {
		oldIdx := oldArch.componentMap[id]
		newIdx := newArch.componentMap[id]
		newArch.data[newIdx][newIndex] = oldArch.data[oldIdx][loc.index]
		oldArch.data[oldIdx][loc.index] = nil // Clear old to avoid memory leak
	}
	// Update entity location
	w.entityInfo[e] = entityLocation{arch: newArch, index: newIndex}
	// Remove from old archetype
	oldArch.swapRemove(loc.index)
	// Update swapped entity's location if necessary
	if oldArch.length > loc.index {
		swappedE := oldArch.entities[loc.index]
		w.entityInfo[swappedE] = entityLocation{arch: oldArch, index: loc.index}
	}
}

// GetComponent retrieves a component from an entity if it exists.
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

// QueryAll returns all entities that have all the specified components.
func (self *World) QueryAll(cts ...ComponentID) []Entity {
	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAll(cts) {
			for i := 0; i < a.length; i++ {
				entities = append(entities, a.entities[i])
			}
		}
	}
	return entities
}

// QueryAny returns all entities that have at least one of the specified components.
func (self *World) QueryAny(cts ...ComponentID) []Entity {
	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAny(cts) {
			for i := 0; i < a.length; i++ {
				entities = append(entities, a.entities[i])
			}
		}
	}
	return entities
}

// DestroyEntity removes an entity and all its components from the world.
func (w *World) DestroyEntity(e Entity) {
	loc, ok := w.entityInfo[e]
	if !ok {
		return // Entity does not exist
	}
	arch := loc.arch
	arch.swapRemove(loc.index)
	// Update swapped entity's location if necessary
	if arch.length > loc.index {
		swappedE := arch.entities[loc.index]
		w.entityInfo[swappedE] = entityLocation{arch: arch, index: loc.index}
	}
	delete(w.entityInfo, e)
}

// DestroyAllEntitiesWith destroys all entities that have all the specified components.
func (w *World) DestroyAllEntitiesWith(cts ...ComponentID) {
	for _, a := range w.archetypes {
		if a.containsAll(cts) {
			// Destroy all entities in this archetype
			for i := a.length - 1; i >= 0; i-- {
				e := a.entities[i]
				a.swapRemove(i)
				delete(w.entityInfo, e)
			}
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
