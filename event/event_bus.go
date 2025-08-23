package event

import (
	"log"
	"reflect"
	"sync"
)

type EventBus struct {
	mu        sync.Mutex
	listeners map[reflect.Type][]func(interface{})
	queue     []interface{}
}

func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[reflect.Type][]func(interface{})),
	}
}

// Subscribe: Pass event type and handler. Handler must accept interface{} and type-assert internally.
func (eb *EventBus) Subscribe(eventType interface{}, handler func(interface{})) {
	t := reflect.TypeOf(eventType)
	eb.mu.Lock()
	eb.listeners[t] = append(eb.listeners[t], handler)
	eb.mu.Unlock()
}

// Publish: Pass any event struct.
func (eb *EventBus) Publish(event interface{}) {
	eb.mu.Lock()
	eb.queue = append(eb.queue, event)
	eb.mu.Unlock()
}

// Process: Call in World.Update() to dispatch events.
func (eb *EventBus) Process() {
	eb.mu.Lock()
	q := eb.queue
	eb.queue = nil
	eb.mu.Unlock()

	for _, event := range q {
		t := reflect.TypeOf(event)
		eb.mu.Lock()
		handlers, ok := eb.listeners[t]
		hcopy := make([]func(interface{}), len(handlers))
		copy(hcopy, handlers)
		eb.mu.Unlock()
		if !ok {
			continue
		}
		for _, h := range hcopy {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("Event handler panic: %v", r)
					}
				}()
				h(event)
			}()
		}
	}
}
