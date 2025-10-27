package event

import "reflect"

// MaxEventTypes defines the maximum number of unique event types that can be
// registered in the EventBus. This value is fixed at 256.
const MaxEventTypes = 256

// EventBus provides a simple, efficient, and type-safe event bus for decoupled
// communication between different parts of an application. It allows systems to
// subscribe to specific event types and publish events to all interested
// listeners without direct dependencies.
//
// The EventBus is designed for high performance, with `Publish` operations being
// allocation-free.
type EventBus struct {
	eventTypeMap    map[reflect.Type]uint8
	handlers        [MaxEventTypes][]interface{}
	nextEventTypeID uint8
}

// Subscribe registers a handler function to be called when an event of type `T`
// is published. Handlers are stored in the order they are subscribed.
//
// This operation may allocate memory if it's the first time subscribing to a
// particular event type or if the internal handler list needs to be resized.
//
// Parameters:
//   - bus: The EventBus instance to subscribe to.
//   - handler: A function that takes a single argument of type `T`.
func Subscribe[T any](bus *EventBus, handler func(T)) {
	t := reflect.TypeFor[T]()
	id := bus.getEventTypeID(t)
	if cap(bus.handlers[id]) == 0 {
		bus.handlers[id] = make([]interface{}, 0, 4) // Preallocate small capacity to reduce reallocs
	}
	bus.handlers[id] = append(bus.handlers[id], handler)
}

// Publish broadcasts an event of type `T` to all registered handlers for that
// type. The handlers are called synchronously in the order they were subscribed.
//
// This operation is highly optimized and is allocation-free, making it suitable
// for performance-critical code paths.
//
// Parameters:
//   - bus: The EventBus instance to publish to.
//   - event: The event data of type `T` to be sent to handlers.
func Publish[T any](bus *EventBus, event T) {
	t := reflect.TypeFor[T]()
	if id, ok := bus.eventTypeMap[t]; ok {
		hs := bus.handlers[id]
		for _, h := range hs {
			h.(func(T))(event)
		}
	}
}

// getEventTypeID retrieves or assigns an ID for the event type.
func (bus *EventBus) getEventTypeID(t reflect.Type) uint8 {
	if bus.eventTypeMap == nil {
		bus.eventTypeMap = make(map[reflect.Type]uint8)
	}
	if id, ok := bus.eventTypeMap[t]; ok {
		return id
	}
	id := bus.nextEventTypeID
	bus.nextEventTypeID++
	if int(id) >= MaxEventTypes {
		panic("ecs: too many event types")
	}
	bus.eventTypeMap[t] = id
	return id
}
