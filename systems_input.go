package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

type InputSystem struct {
	bindinds map[Action][]KeyConfig
	filter   *teishoku.Filter[InputComponent]
}

func NewInputSystem(bindinds map[Action][]KeyConfig) *InputSystem {
	return &InputSystem{
		bindinds: bindinds,
	}
}

func (self *InputSystem) Initialize(w *teishoku.World) {
	self.filter = self.filter.New(w)
}

func (self *InputSystem) Update(w *teishoku.World, dt float64) {
	// Get mouse wheel deltas once per frame
	wx, wy := ebiten.Wheel()

	self.filter.Reset()
	for self.filter.Next() {
		inp := self.filter.Get()

		// Reset states for the current frame
		for action := range inp.Bindings {
			inp.JustPressed[action] = false
			inp.JustReleased[action] = false
		}

		// set the mouse wheel deltas
		inp.MouseWheelX = wx
		inp.MouseWheelY = wy

		// Now, check for the current state for all bindings.
		for action := range inp.Bindings {
			// A single action can be triggered by multiple bindings (e.g., keyboard and gamepad)
			// We use a logical OR to ensure that if any binding is met, the action is triggered.
			isAnyJustPressed := false
			isAnyPressed := false
			isAnyJustReleased := false

			for _, binding := range inp.Bindings[action] {
				modsDown := true
				for _, mod := range binding.Modifiers {
					if !isPressed(inp.ID, mod) {
						modsDown = false
						break
					}
				}

				if modsDown {
					if isJustPressed(inp.ID, binding.Primary) {
						isAnyJustPressed = true
					}
					if isPressed(inp.ID, binding.Primary) {
						isAnyPressed = true
					}
					if isJustReleased(inp.ID, binding.Primary) {
						isAnyJustReleased = true
					}
				}
			}

			// Update the component's state based on the calculated values
			inp.JustPressed[action] = isAnyJustPressed
			inp.JustReleased[action] = isAnyJustReleased
			inp.Pressed[action] = isAnyPressed
		}
	}
}
