package katsu2d

import (
	"reflect"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/edwinsyarief/katsu2d/event"
)

const (
	event_bus = "EventBus"
)

// Entity is a unique identifier for an entity, including a version for safety.
type Entity struct {
	ID      uint32
	Version uint32
}

// entityMeta stores internal metadata for each entity.
type entityMeta struct {
	Archetype *Archetype
	Index     uint32
	Version   uint32 // Added Version field for correct validation
}

// World manages entities and their components.
type World struct {
	mu           sync.RWMutex // A read/write mutex for better concurrency performance.
	nextEntityID uint32
	// We'll store entity metadata in a map for proper sparse ID handling.
	entities map[uint32]entityMeta
	// The archetypes map now maps a component mask to a single archetype struct.
	archetypes map[uint64]*Archetype
	// A map to store component pools for each component type to reduce GC.
	componentPools sync.Map
	// An array to store entities that need to be removed
	toRemove []Entity
	// Resources storage
	Resources sync.Map
}

// Archetype is a contiguous block of memory for a specific set of components.
type Archetype struct {
	mask     uint64
	entities []Entity
	// Component data is stored in a slice of slices, where each inner slice
	// holds all components of a single type. This is the key to cache efficiency.
	componentData [][]any
	// componentMap maps a component ID to its index in the componentData slice.
	componentMap map[ComponentID]int
	// componentIDs are sorted to ensure consistent archetype masks.
	componentIDs []ComponentID
}

// NewWorld creates a new, high-performance ECS world.
func NewWorld() *World {
	w := &World{
		nextEntityID: 1,
		entities:     make(map[uint32]entityMeta), // Initialized as a map
		archetypes:   make(map[uint64]*Archetype),
		toRemove:     make([]Entity, 0),
		//Resources:    sync.Map{},
	}
	// Create the initial empty archetype.
	w.archetypes[0] = &Archetype{
		mask:          0,
		entities:      make([]Entity, 0, 1024),
		componentData: make([][]any, 0),
		componentMap:  make(map[ComponentID]int),
		componentIDs:  make([]ComponentID, 0),
	}
	return w
}

func (self *World) initEventBus() *event.EventBus {
	if eb, ok := self.Resources.Load(event_bus); ok {
		return eb.(*event.EventBus)
	}

	eb := event.NewEventBus()
	self.Resources.Store(event_bus, eb)
	return eb
}

func (self *World) GetEventBus() *event.EventBus {
	if eb, ok := self.Resources.Load(event_bus); ok {
		return eb.(*event.EventBus)
	}

	return nil
}

func (self *World) ProcessEventBus() {
	if eb, ok := self.Resources.Load(event_bus); ok {
		eb.(*event.EventBus).Process()
	}
}

// getOrCreateArchetype finds an existing archetype for the given mask, or creates a new one.
func (self *World) getOrCreateArchetype(mask uint64) *Archetype {
	if arch, ok := self.archetypes[mask]; ok {
		return arch
	}

	// New archetype doesn't exist, create it.
	newArch := &Archetype{
		mask:          mask,
		entities:      make([]Entity, 0, 1024),
		componentData: make([][]any, 0),
		componentMap:  make(map[ComponentID]int),
		componentIDs:  make([]ComponentID, 0),
	}
	self.archetypes[mask] = newArch

	// Populate the new archetype with component data slices based on the mask.
	compIDs := make([]ComponentID, 0)
	for id := range componentTypes {
		if (mask>>uint64(id))&1 == 1 {
			compIDs = append(compIDs, id)
		}
	}

	// Sort component IDs for consistent memory layout.
	sort.Slice(compIDs, func(i, j int) bool {
		return compIDs[i] < compIDs[j]
	})

	newArch.componentIDs = compIDs
	newArch.componentData = make([][]any, len(compIDs))

	for i, id := range compIDs {
		newArch.componentMap[id] = i
		newArch.componentData[i] = make([]any, 0, 1024)
	}

	return newArch
}

// CreateEntity creates a new entity.
func (self *World) CreateEntity() Entity {
	self.mu.Lock()
	defer self.mu.Unlock()

	id := atomic.AddUint32(&self.nextEntityID, 1) - 1
	e := Entity{ID: id, Version: 1} // Initial version is 1.

	// Add the new entity to the archetype for mask 0
	arch := self.archetypes[0]
	self.entities[id] = entityMeta{Archetype: arch, Index: uint32(len(arch.entities)), Version: e.Version}
	arch.entities = append(arch.entities, e)

	return e
}

// RemoveEntity marks an entity for removal.
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

	removeSet := make(map[Entity]struct{})
	for _, e := range self.toRemove {
		meta, ok := self.entities[e.ID]
		if ok && e.Version == meta.Version {
			removeSet[e] = struct{}{}
		}
	}

	for e := range removeSet {
		meta := self.entities[e.ID]
		arch := meta.Archetype

		// Return components to their pools
		for i, compID := range arch.componentIDs {
			compType := componentTypes[compID]
			comp := arch.componentData[i][meta.Index]

			// Use a pool for the specific component type
			pool, _ := self.componentPools.LoadOrStore(compType, &sync.Pool{
				New: func() any {
					return reflect.New(compType.Elem()).Interface()
				},
			})
			pool.(*sync.Pool).Put(comp)
		}

		// Remove the entity from its archetype and update metadata
		self.removeEntityFromArchetype(arch, e, meta.Index)

		// Invalidate the entity by deleting its metadata and incrementing its version.
		delete(self.entities, e.ID)
		e.Version++ // Increment version for safety
	}
	self.toRemove = self.toRemove[:0]
}

// removeEntityFromArchetype is a helper function to remove an entity and its components from an archetype.
func (self *World) removeEntityFromArchetype(arch *Archetype, e Entity, index uint32) {
	// Swap the entity with the last one in the archetype's entity list.
	lastIndex := len(arch.entities) - 1
	lastEntity := arch.entities[lastIndex]
	arch.entities[index] = lastEntity

	// Update the meta for the moved entity.
	if lastEntity.ID != e.ID { // Don't update if we're removing the last entity itself
		meta := self.entities[lastEntity.ID]
		meta.Index = index
		self.entities[lastEntity.ID] = meta
	}

	// Now truncate the slice.
	arch.entities = arch.entities[:lastIndex]

	// Also swap and truncate the component data for all component types.
	for i := range arch.componentData {
		lastComp := arch.componentData[i][lastIndex]
		arch.componentData[i][index] = lastComp
		arch.componentData[i] = arch.componentData[i][:lastIndex]
	}
}

// AddComponent adds a component to an entity.
func (self *World) AddComponent(e Entity, comp any) {
	self.mu.Lock()
	defer self.mu.Unlock()

	meta, ok := self.entities[e.ID]
	if !ok || meta.Archetype == nil || e.Version != meta.Version {
		return // Invalid or removed entity.
	}

	compType := reflect.TypeOf(comp)
	compID, ok := typeToID[compType]
	if !ok {
		panic("Component not registered: " + compType.Name())
	}

	oldArch := meta.Archetype
	oldIndex := meta.Index

	// Calculate the new archetype mask.
	newMask := oldArch.mask | (1 << uint64(compID))

	// Find or create the new archetype.
	newArch := self.getOrCreateArchetype(newMask)

	// Add the entity to the new archetype.
	newIndex := len(newArch.entities)
	newArch.entities = append(newArch.entities, e)

	// Copy existing components from the old archetype to the new one.
	for _, oldCompID := range oldArch.componentIDs {
		oldCompSliceIdx := oldArch.componentMap[oldCompID]
		newCompSliceIdx := newArch.componentMap[oldCompID]

		oldComp := oldArch.componentData[oldCompSliceIdx][oldIndex]

		// DEEP COPY THE COMPONENT to avoid pointer aliasing.
		newComp := reflect.New(reflect.TypeOf(oldComp).Elem()).Interface()
		reflect.ValueOf(newComp).Elem().Set(reflect.ValueOf(oldComp).Elem())

		newArch.componentData[newCompSliceIdx] = append(newArch.componentData[newCompSliceIdx], newComp)
	}

	// Add the new component to the new archetype.
	newCompSliceIdx := newArch.componentMap[compID]
	newArch.componentData[newCompSliceIdx] = append(newArch.componentData[newCompSliceIdx], comp)

	// Update entity metadata.
	self.entities[e.ID] = entityMeta{Archetype: newArch, Index: uint32(newIndex), Version: e.Version}

	// Remove from old archetype (this is the key to performance).
	self.removeEntityFromArchetype(oldArch, e, oldIndex)
}

// GetComponent retrieves a component from an entity using a ComponentID.
func (self *World) GetComponent(e Entity, id ComponentID) (any, bool) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	meta, ok := self.entities[e.ID]
	if !ok || meta.Archetype == nil || e.Version != meta.Version {
		return nil, false // Invalid or removed entity.
	}

	arch := meta.Archetype
	if compIndex, ok := arch.componentMap[id]; ok {
		comp := arch.componentData[compIndex][meta.Index]
		return comp, true
	}

	return nil, false
}

// RemoveComponent removes a component from an entity.
func (self *World) RemoveComponent(e Entity, id ComponentID) {
	self.mu.Lock()
	defer self.mu.Unlock()

	meta, ok := self.entities[e.ID]
	if !ok || meta.Archetype == nil || e.Version != meta.Version {
		return // Invalid or removed entity.
	}

	oldArch := meta.Archetype
	oldIndex := meta.Index

	// Return the component being removed to its pool.
	oldComp := oldArch.componentData[oldArch.componentMap[id]][oldIndex]
	compType := reflect.TypeOf(oldComp)
	pool, _ := self.componentPools.LoadOrStore(compType, &sync.Pool{
		New: func() any {
			return reflect.New(compType.Elem()).Interface()
		},
	})
	pool.(*sync.Pool).Put(oldComp)

	// Calculate new mask.
	newMask := oldArch.mask & ^(1 << uint64(id))

	// Find or create the new archetype.
	newArch := self.getOrCreateArchetype(newMask)

	// Add the entity to the new archetype.
	newIndex := len(newArch.entities)
	newArch.entities = append(newArch.entities, e)

	// Copy components that are NOT being removed.
	for _, oldCompID := range oldArch.componentIDs {
		if oldCompID == id {
			continue
		}
		oldCompSliceIdx := oldArch.componentMap[oldCompID]
		newCompSliceIdx := newArch.componentMap[oldCompID]

		oldComp := oldArch.componentData[oldCompSliceIdx][oldIndex]

		// DEEP COPY THE COMPONENT
		newComp := reflect.New(reflect.TypeOf(oldComp).Elem()).Interface()
		reflect.ValueOf(newComp).Elem().Set(reflect.ValueOf(oldComp).Elem())

		newArch.componentData[newCompSliceIdx] = append(newArch.componentData[newCompSliceIdx], newComp)
	}

	// Update entity metadata.
	self.entities[e.ID] = entityMeta{Archetype: newArch, Index: uint32(newIndex), Version: e.Version}

	// Remove from old archetype.
	self.removeEntityFromArchetype(oldArch, e, oldIndex)
}

// Query returns entities that have all specified components.
func (self *World) Query(componentIDs ...ComponentID) []Entity {
	self.mu.RLock()
	defer self.mu.RUnlock()

	res := make([]Entity, 0)
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

// QueryExact returns entities that have exactly the specified components (no more, no less).
func (self *World) QueryExact(componentIDs ...ComponentID) []Entity {
	self.mu.RLock()
	defer self.mu.RUnlock()

	res := make([]Entity, 0)
	var mask uint64
	for _, id := range componentIDs {
		mask |= 1 << uint64(id)
	}

	// Only return entities from archetypes that match the exact mask
	if arch, exists := self.archetypes[mask]; exists {
		res = append(res, arch.entities...)
	}

	return res
}

// QueryWithExclusion returns entities that have any components from includes
// but none of the components from excludes.
func (self *World) QueryWithExclusion(includes []ComponentID, excludes []ComponentID) []Entity {
	self.mu.RLock()
	defer self.mu.RUnlock()

	res := make([]Entity, 0)

	// Create masks for include and exclude components
	var includeMask uint64
	for _, id := range includes {
		includeMask |= 1 << uint64(id)
	}

	var excludeMask uint64
	for _, id := range excludes {
		excludeMask |= 1 << uint64(id)
	}

	// Check each archetype
	for archMask, arch := range self.archetypes {
		// Check if archetype has ANY of the included components
		// AND NONE of the excluded components
		if (archMask&includeMask != 0) && (archMask&excludeMask == 0) {
			res = append(res, arch.entities...)
		}
	}

	return res
}

// QueryTag queries all entities which has a tag component and whose tag value is available in tags
func (self *World) QueryTag(tags ...string) []Entity {
	self.mu.RLock()
	defer self.mu.RUnlock()

	res := make([]Entity, 0)
	tagSet := make(map[string]struct{})
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	tagQuery := self.Query(CTTag)

	for _, entity := range tagQuery {
		if component, ok := self.GetComponent(entity, CTTag); ok {
			if tagComponent, ok := component.(*TagComponent); ok {
				if _, ok := tagSet[tagComponent.Tag]; ok {
					res = append(res, entity)
				}
			}
		}
	}

	return res
}

// QueryExclude queries all entities that has one of component, but without tag or tag is not in the list
func (self *World) QueryWithTagExclusion(componentIDs []ComponentID, excludedTags ...string) []Entity {
	self.mu.RLock()
	defer self.mu.RUnlock()

	res := make([]Entity, 0)
	tagSet := make(map[string]struct{})
	for _, tag := range excludedTags {
		tagSet[tag] = struct{}{}
	}

	var includeMask uint64
	for _, id := range componentIDs {
		includeMask |= 1 << uint64(id)
	}

	for archMask, arch := range self.archetypes {
		if archMask&includeMask != 0 {
			// Archetype has at least one of the included components.
			// Now, check for tag exclusion.
			if (archMask & (1 << uint64(CTTag))) == 0 {
				// Archetype does not have a tag component, so all entities are included.
				res = append(res, arch.entities...)
			} else {
				// Archetype has a tag component, so we need to check each entity.
				for _, entity := range arch.entities {
					if component, ok := self.GetComponent(entity, CTTag); ok {
						if tagComponent, ok := component.(*TagComponent); ok {
							if _, ok := tagSet[tagComponent.Tag]; !ok {
								// Tag is not in the exclusion list, so include the entity.
								res = append(res, entity)
							}
						}
					}
				}
			}
		}
	}

	return res
}
