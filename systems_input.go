package katsu2d

import (
	"github.com/edwinsyarief/lazyecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// InputSystem is responsible for updating the state of all InputComponents.
type InputSystem struct{}

// NewInputSystem creates and initializes an InputSystem.
func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

// Update processes all input components in the world.
func (s *InputSystem) Update(world *lazyecs.World, delta float64) {
	// Query all entities with an input component
	query := world.Query(CTInput)

	// Get mouse wheel deltas once per frame
	wx, wy := ebiten.Wheel()

	for query.Next() {
		for _, entity := range query.Entities() {
			comp, _ := lazyecs.GetComponent[InputComponent](world, entity)
			// Reset states for the current frame
			for action := range comp.Bindings {
				comp.JustPressed[action] = false
				comp.JustReleased[action] = false
			}

			// set the mouse wheel deltas
			comp.MouseWheelX = wx
			comp.MouseWheelY = wy

			// Now, check for the current state for all bindings.
			for action := range comp.Bindings {
				// A single action can be triggered by multiple bindings (e.g., keyboard and gamepad)
				// We use a logical OR to ensure that if any binding is met, the action is triggered.
				isAnyJustPressed := false
				isAnyPressed := false
				isAnyJustReleased := false

				for _, binding := range comp.Bindings[action] {
					modsDown := true
					for _, mod := range binding.Modifiers {
						if !isPressed(mod) {
							modsDown = false
							break
						}
					}

					if modsDown {
						if isJustPressed(binding.Primary) {
							isAnyJustPressed = true
						}
						if isPressed(binding.Primary) {
							isAnyPressed = true
						}
						if isJustReleased(binding.Primary) {
							isAnyJustReleased = true
						}
					}
				}

				// Update the component's state based on the calculated values
				comp.JustPressed[action] = isAnyJustPressed
				comp.JustReleased[action] = isAnyJustReleased
				comp.Pressed[action] = isAnyPressed
			}
		}
	}
}
