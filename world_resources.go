package katsu2d

import (
	"github.com/edwinsyarief/katsu2d/event"
	"github.com/edwinsyarief/lazyecs"
)

const (
	event_bus = "event_bus"
)

func GetEventBus(world *lazyecs.World) *event.EventBus {
	ebAny, _ := world.Resources.LoadOrStore(event_bus, event.NewEventBus())
	eb := ebAny.(*event.EventBus)
	return eb
}

func ProcessEventBus(world *lazyecs.World) {
	eb := GetEventBus(world)
	eb.Process()
}
