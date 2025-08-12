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
	// componentTypeMap maps component IDs back to their types for debugging.
	componentTypeMap = make(map[ComponentID]reflect.Type)
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
// This function should be called once per component type at program startup.
func RegisterComponent[T any]() ComponentID {
	// We get the type of T, not a pointer to T. This ensures consistency.
	t := reflect.TypeOf((*T)(nil)).Elem()
	if id, ok := componentIDMap[t]; ok {
		return id // Component already registered, return its ID.
	}
	// Assign a new ID and increment the counter.
	id := nextComponentID
	nextComponentID++
	componentIDMap[t] = id
	componentTypeMap[id] = t
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
	// We're now using a map from ComponentID to a slice of reflect.Value,
	// which is safer and more robust than a generic `[][]any`.
	data     map[ComponentID]reflect.Value // The dense arrays of component data.
	capacity int                           // The current allocated capacity.
	length   int                           // The number of active entities.
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

// containsAny checks if the archetype contains at least one of the specified components.
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

		for id, oldData := range self.data {
			// Create a new slice with the new capacity.
			newData := reflect.MakeSlice(oldData.Type(), self.length, newCap)
			// Copy the old data to the new slice.
			reflect.Copy(newData, oldData)
			self.data[id] = newData
		}

		self.capacity = newCap
	}

	idx := self.length
	self.length++
	return idx
}

// pendingChange represents a structural change to be processed later
type pendingChange struct {
	entity      Entity
	component   any
	operation   changeOperation
	componentID ComponentID
}

type changeOperation int

const (
	opAddComponent changeOperation = iota
	opRemoveComponent
)

// World manages all entities, archetypes, and systems.
type World struct {
	nextEntity      Entity                    // Next available entity ID.
	archetypes      map[string]*Archetype     // Map of archetype keys to archetypes.
	entityInfo      map[Entity]entityLocation // Map of entity to its location.
	systems         []System                  // A slice to hold all registered systems.
	pendingChanges  []pendingChange           // Queue for structural changes during query processing
	processingQuery bool                      // Flag to prevent structural changes during queries
}

// NewWorld creates a new ECS world with an empty archetype.
func NewWorld() *World {
	w := &World{
		archetypes:     make(map[string]*Archetype),
		entityInfo:     make(map[Entity]entityLocation),
		systems:        make([]System, 0),
		pendingChanges: make([]pendingChange, 0),
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
	// Process any pending structural changes after all systems have run
	self.processPendingChanges()
}

// processPendingChanges processes all queued structural changes
func (self *World) processPendingChanges() {
	if len(self.pendingChanges) == 0 {
		return
	}

	changes := self.pendingChanges
	self.pendingChanges = self.pendingChanges[:0] // Clear the slice

	for _, change := range changes {
		switch change.operation {
		case opAddComponent:
			// Validate entity still exists and doesn't already have the component
			if loc, ok := self.entityInfo[change.entity]; ok {
				ct, err := getComponentID(change.component)
				if err != nil {
					fmt.Printf("Failed to get component ID for pending change: %v\n", err)
					continue
				}
				if !loc.arch.has(ct) {
					newIDs := append(loc.arch.componentIDs, ct)
					newArch := self.getOrCreateArchetype(newIDs)
					self.migrate(change.entity, loc.arch, newArch, change.component)
				}
			}
		case opRemoveComponent:
			if loc, ok := self.entityInfo[change.entity]; ok {
				if loc.arch.has(change.componentID) {
					var newIDs []ComponentID
					for _, id := range loc.arch.componentIDs {
						if id != change.componentID {
							newIDs = append(newIDs, id)
						}
					}
					newArch := self.getOrCreateArchetype(newIDs)
					self.migrate(change.entity, loc.arch, newArch, nil)
				}
			}
		}
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
		data:         make(map[ComponentID]reflect.Value),
		capacity:     initialCap,
		length:       0,
	}

	for i, id := range ids {
		a.componentMap[id] = i
		// Get the type of the component from our global map
		t, ok := componentTypeMap[id]
		if !ok {
			panic(fmt.Sprintf("Component ID %d not registered", id))
		}
		// Create a slice of this component type. `reflect.New` creates a pointer,
		// so we must use `reflect.New(reflect.SliceOf(t)).Elem()` to get the slice value itself.
		sliceType := reflect.SliceOf(t)
		a.data[id] = reflect.MakeSlice(sliceType, initialCap, initialCap)
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
// It is now more robust to handle both pointers and non-pointers.
// It also returns an error instead of panicking.
func getComponentID(c any) (ComponentID, error) {
	if c == nil {
		return 0, fmt.Errorf("component is nil")
	}

	t := reflect.TypeOf(c)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	id, ok := componentIDMap[t]
	if !ok {
		return 0, fmt.Errorf("unregistered component type: %s. Call RegisterComponent first", t.String())
	}
	return id, nil
}

// AddComponent adds a component to an entity. This moves the entity
// to a new, more specific archetype.
// Queues changes during query processing to prevent iteration invalidation
func (self *World) AddComponent(e Entity, c any) {
	if c == nil {
		return
	}

	if self.processingQuery {
		self.pendingChanges = append(self.pendingChanges, pendingChange{
			entity:    e,
			component: c,
			operation: opAddComponent,
		})
		return
	}

	ct, err := getComponentID(c)
	if err != nil {
		panic(err.Error())
	}

	loc, ok := self.entityInfo[e]
	if !ok {
		panic("Entity does not exist")
	}

	oldArch := loc.arch
	if oldArch.has(ct) {
		fmt.Printf("Entity %d already has component %d. Skipping.\n", e, ct)
		return
	}

	newIDs := append(oldArch.componentIDs, ct)
	newArch := self.getOrCreateArchetype(newIDs)

	self.migrate(e, oldArch, newArch, c)
}

// RemoveComponents removes one or more components from an entity.
// Queues changes during query processing to prevent iteration invalidation
func (self *World) RemoveComponents(e Entity, cts ...ComponentID) {
	if self.processingQuery {
		for _, ct := range cts {
			self.pendingChanges = append(self.pendingChanges, pendingChange{
				entity:      e,
				operation:   opRemoveComponent,
				componentID: ct,
			})
		}
		return
	}

	loc, ok := self.entityInfo[e]
	if !ok {
		return
	}

	oldArch := loc.arch
	removeSet := make(map[ComponentID]bool, len(cts))
	for _, ct := range cts {
		if oldArch.has(ct) {
			removeSet[ct] = true
		}
	}
	if len(removeSet) == 0 {
		return
	}

	var newIDs []ComponentID
	for _, id := range oldArch.componentIDs {
		if !removeSet[id] {
			newIDs = append(newIDs, id)
		}
	}

	newArch := self.getOrCreateArchetype(newIDs)
	self.migrate(e, oldArch, newArch, nil)
}

// swapAndRemove is an internal helper function that removes an entity from an archetype
// at a given index. It swaps the last element into the removed position to
// maintain a dense array. This function now performs all necessary state updates
// for the swapped entity.
func (self *World) swapAndRemove(arch *Archetype, index int) {
	if arch.length == 0 || index >= arch.length {
		return
	}

	lastIndex := arch.length - 1

	if index != lastIndex {
		swappedEntity := arch.entities[lastIndex]
		arch.entities[index] = swappedEntity

		if loc, ok := self.entityInfo[swappedEntity]; ok {
			loc.index = index
			self.entityInfo[swappedEntity] = loc
		}

		for _, data := range arch.data {
			// Copy the last element's data to the removed index
			data.Index(index).Set(data.Index(lastIndex))
		}
	}

	// For memory safety, we clear the last element.
	for _, data := range arch.data {
		data.Index(lastIndex).Set(reflect.Zero(data.Type().Elem()))
	}
	arch.entities[lastIndex] = 0

	arch.length--
}

// migrate performs the atomic movement of an entity from one archetype to another.
// It is now more robust and type-safe using reflection.
func (self *World) migrate(e Entity, oldArch, newArch *Archetype, newComponent any) {
	loc := self.entityInfo[e]

	newIndex := newArch.addEmptySlot()
	newArch.entities[newIndex] = e

	// Copy existing components from the old archetype to the new one.
	for _, oldID := range oldArch.componentIDs {
		// Check if the component is also in the new archetype.
		if _, ok := newArch.componentMap[oldID]; ok {
			oldData := oldArch.data[oldID]
			newData := newArch.data[oldID]

			// Defensive check to prevent "call of reflect.Value.Index on zero Value"
			if !oldData.IsValid() || !newData.IsValid() {
				// This should not happen if the archetype is valid.
				// For now, we'll just skip the copy and continue.
				continue
			}

			// Copy the component instance from the old archetype to the new one.
			newData.Index(newIndex).Set(oldData.Index(loc.index))
		}
	}

	// Add the new component if one exists.
	if newComponent != nil {
		ct, _ := getComponentID(newComponent)
		if _, ok := newArch.componentMap[ct]; ok {
			newData := newArch.data[ct]

			// Handle both pointers and values for assignment.
			compVal := reflect.ValueOf(newComponent)
			if compVal.IsValid() && compVal.Kind() == reflect.Ptr {
				compVal = compVal.Elem()
			}

			if newData.IsValid() && compVal.IsValid() {
				newData.Index(newIndex).Set(compVal)
			}
		}
	}

	self.swapAndRemove(oldArch, loc.index)
	self.entityInfo[e] = entityLocation{arch: newArch, index: newIndex}
}

// GetComponent retrieves a component from an entity.
// It now returns a pointer to the component for consistency with the user's factories.
func (self *World) GetComponent(e Entity, ct ComponentID) any {
	loc, ok := self.entityInfo[e]
	if !ok {
		return nil
	}
	if !loc.arch.has(ct) {
		return nil
	}
	// Get the component slice using the ID.
	compSlice := loc.arch.data[ct]
	if compSlice.IsValid() && loc.index < compSlice.Len() {
		// Extract the component value and return a pointer to it.
		// This makes the API consistent with what the factories expect.
		val := compSlice.Index(loc.index)
		return val.Addr().Interface()
	}
	return nil
}

// GetAllEntityComponents retrieves all components for a given entity,
// returning them as a map from component ID to the component instance pointer.
// This is now consistent with the user's expectations.
func (self *World) GetAllEntityComponents(e Entity) map[ComponentID]any {
	loc, ok := self.entityInfo[e]
	if !ok {
		return nil
	}

	components := make(map[ComponentID]any, len(loc.arch.componentIDs))
	for _, ct := range loc.arch.componentIDs {
		compSlice := loc.arch.data[ct]
		if compSlice.IsValid() && loc.index < compSlice.Len() {
			// Return a pointer to the component value, not the value itself.
			val := compSlice.Index(loc.index)
			components[ct] = val.Addr().Interface()
		}
	}
	return components
}

// GetComponentSafe retrieves a component from an entity with explicit nil checking.
// Returns a pointer to the component and a boolean indicating if the component exists and is not nil.
func (self *World) GetComponentSafe(e Entity, ct ComponentID) (any, bool) {
	loc, ok := self.entityInfo[e]
	if !ok {
		return nil, false
	}
	if !loc.arch.has(ct) {
		return nil, false
	}
	compSlice := loc.arch.data[ct]
	if compSlice.IsValid() && loc.index < compSlice.Len() {
		val := compSlice.Index(loc.index)
		component := val.Addr().Interface()
		return component, component != nil
	}
	return nil, false
}

// QueryAll returns all entities that have ALL of the specified components.
// Sets processingQuery flag to prevent structural changes during iteration
func (self *World) QueryAll(cts ...ComponentID) []Entity {
	self.processingQuery = true
	defer func() { self.processingQuery = false }()

	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAll(cts) {
			entities = append(entities, a.entities[:a.length]...)
		}
	}
	return entities
}

// QueryAny returns all entities that have AT LEAST ONE of the specified components.
// Sets processingQuery flag to prevent structural changes during iteration
func (self *World) QueryAny(cts ...ComponentID) []Entity {
	self.processingQuery = true
	defer func() { self.processingQuery = false }()

	var entities []Entity
	for _, a := range self.archetypes {
		if a.containsAny(cts) {
			entities = append(entities, a.entities[:a.length]...)
		}
	}
	return entities
}

// QueryWith returns entities that have ALL required components AND at least one from optional components
func (self *World) QueryWith(required []ComponentID, optional []ComponentID) []Entity {
	self.processingQuery = true
	defer func() { self.processingQuery = false }()

	var entities []Entity
	for _, a := range self.archetypes {
		hasRequired := len(required) == 0 || a.containsAll(required)
		hasOptional := len(optional) == 0 || a.containsAny(optional)
		if hasRequired && hasOptional {
			entities = append(entities, a.entities[:a.length]...)
		}
	}
	return entities
}

// DestroyEntity removes an entity and all its components from the world.
func (self *World) DestroyEntity(e Entity) {
	loc, ok := self.entityInfo[e]
	if !ok {
		return
	}

	arch := loc.arch
	self.swapAndRemove(arch, loc.index)

	delete(self.entityInfo, e)
}

// DestroyAllEntitiesWith destroys all entities that have all the specified components.
func (self *World) DestroyAllEntitiesWith(cts ...ComponentID) {
	for _, a := range self.archetypes {
		if a.containsAll(cts) {
			for i := 0; i < a.length; i++ {
				e := a.entities[i]
				delete(self.entityInfo, e)
			}
			a.length = 0
		}
	}
}

// DestroyAllEntities removes all entities from the world, resetting it to empty.
func (self *World) DestroyAllEntities() {
	for _, a := range self.archetypes {
		a.length = 0
	}
	self.entityInfo = make(map[Entity]entityLocation)
	self.nextEntity = 0
}
