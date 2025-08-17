package katsu2d

import "github.com/hajimehoshi/ebiten/v2"

// InputSystem is an UpdateSystem that handles all game input.
type InputSystem struct{}

// NewInputSystem creates a new input system
func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

// Update implements the UpdateSystem interface. It polls the keyboard
// and gamepad and updates the internal state of all actions. This should be run
// once per game tick.
func (self *InputSystem) Update(world *World, dt float64) {
	entities := world.Query(CTInput)
	for _, e := range entities {
		comp, _ := world.GetComponent(e, CTInput)
		inputComp := comp.(*InputComponent)

		// First, copy the current state to the previous state.
		for action, isPressed := range inputComp.actionState {
			inputComp.previousState[action] = isPressed
		}

		// Then, clear the current state to be re-evaluated.
		for action := range inputComp.actionState {
			inputComp.actionState[action] = false
		}

		// Iterate through all defined actions and their bindings to check for key presses.
		for action, configs := range inputComp.bindings {
			for _, config := range configs {
				isPressed := false

				// Check for keyboard input if a key is defined.
				if config.Key != ebiten.Key(-1) {
					isPressed = ebiten.IsKeyPressed(config.Key)
					// If the main key is pressed, check for modifiers.
					if isPressed && len(config.Modifiers) > 0 {
						for _, mod := range config.Modifiers {
							if !ebiten.IsKeyPressed(mod) {
								isPressed = false
								break
							}
						}
					}
				}

				// Check for gamepad input if a button is defined.
				if config.GamepadButton != ebiten.GamepadButton(-1) {
					// We assume the first detected gamepad is the one being used.
					// A more advanced system could handle multiple gamepads.
					for _, gID := range ebiten.AppendGamepadIDs(nil) {
						isGamepadButtonDown := ebiten.IsGamepadButtonPressed(gID, config.GamepadButton)
						// If the main button is pressed, check for modifiers.
						if isGamepadButtonDown {
							hasAllModifiers := true
							for _, mod := range config.GamepadModifiers {
								if !ebiten.IsGamepadButtonPressed(gID, mod) {
									hasAllModifiers = false
									break
								}
							}
							if hasAllModifiers {
								isPressed = true
								break // Found a valid gamepad binding, exit this inner loop
							}
						}
					}
				}

				// Check for mouse input if a button is defined.
				if config.MouseButton != ebiten.MouseButton(-1) {
					isMouseButtonDown := ebiten.IsMouseButtonPressed(config.MouseButton)
					// If the main mouse button is pressed, check for modifiers.
					if isMouseButtonDown {
						hasAllModifiers := true
						for _, mod := range config.MouseButtonModifiers {
							if !ebiten.IsMouseButtonPressed(mod) {
								hasAllModifiers = false
								break
							}
						}
						if hasAllModifiers {
							isPressed = true
							break // Found a valid mouse binding, exit this inner loop
						}
					}
				}

				// If any binding for this action is pressed, mark the action as active.
				if isPressed {
					inputComp.actionState[action] = true
					break // Stop checking other key configs for this action.
				}
			}

			// Update the "just pressed" and "just released" states.
			// An action is "just pressed" if it's currently pressed but was not pressed last frame.
			// An action is "just released" if it's currently not pressed but was pressed last frame.
			inputComp.justPressed[action] = inputComp.actionState[action] && !inputComp.previousState[action]
			inputComp.justReleased[action] = !inputComp.actionState[action] && inputComp.previousState[action]
		}
	}

}
