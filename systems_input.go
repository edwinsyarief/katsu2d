package katsu2d

import "github.com/hajimehoshi/ebiten/v2"

type InputSystem struct{}

func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

func (self *InputSystem) Update(world *World, _ float64) {
	entities := world.Query(CTInput)
	for _, e := range entities {
		comp, _ := world.GetComponent(e, CTInput)
		inputComp := comp.(*InputComponent)

		self.updateInputStates(inputComp)
		self.processBindings(inputComp)
		self.updateActionStates(inputComp)
	}
}

func (self *InputSystem) updateInputStates(input *InputComponent) {
	// Copy current to previous state
	for action, isPressed := range input.actionState {
		input.previousState[action] = isPressed
		// Clear current states
		input.actionState[action] = false
		input.wheelState[action] = false
	}
	input.wheelDeltaX, input.wheelDeltaY = ebiten.Wheel()
}

func (self *InputSystem) processBindings(input *InputComponent) {
	for action, configs := range input.bindings {
		for _, config := range configs {
			if self.isBindingActive(config, input) {
				input.actionState[action] = true
				if config.Wheel != WheelNone {
					input.wheelState[action] = true
				}
				break // Stop checking other configs for this action
			}
		}
	}
}

func (self *InputSystem) isBindingActive(config KeyConfig, input *InputComponent) bool {
	return self.isKeyboardBindingActive(config) ||
		self.isGamepadBindingActive(config) ||
		self.isMouseBindingActive(config) ||
		self.isWheelBindingActive(config, input)
}

func (self *InputSystem) checkModifiers(modifiers []ebiten.Key) bool {
	if len(modifiers) == 0 {
		return true
	}
	for _, mod := range modifiers {
		if !ebiten.IsKeyPressed(mod) {
			return false
		}
	}
	return true
}

func (self *InputSystem) checkMouseModifiers(modifiers []ebiten.MouseButton) bool {
	if len(modifiers) == 0 {
		return true
	}
	for _, mod := range modifiers {
		if !ebiten.IsMouseButtonPressed(mod) {
			return false
		}
	}
	return true
}

func (self *InputSystem) isKeyboardBindingActive(config KeyConfig) bool {
	return config.Key != ebiten.Key(-1) &&
		ebiten.IsKeyPressed(config.Key) &&
		self.checkModifiers(config.Modifiers)
}

func (self *InputSystem) isGamepadBindingActive(config KeyConfig) bool {
	if config.GamepadButton == ebiten.GamepadButton(-1) {
		return false
	}

	for _, gID := range ebiten.AppendGamepadIDs(nil) {
		if ebiten.IsGamepadButtonPressed(gID, config.GamepadButton) {
			hasAllModifiers := true
			for _, mod := range config.GamepadModifiers {
				if !ebiten.IsGamepadButtonPressed(gID, mod) {
					hasAllModifiers = false
					break
				}
			}
			if hasAllModifiers {
				return true
			}
		}
	}
	return false
}

func (self *InputSystem) isMouseBindingActive(config KeyConfig) bool {
	return config.MouseButton != ebiten.MouseButton(-1) &&
		ebiten.IsMouseButtonPressed(config.MouseButton) &&
		self.checkMouseModifiers(config.MouseButtonModifiers)
}

func (self *InputSystem) isWheelBindingActive(config KeyConfig, input *InputComponent) bool {
	if config.Wheel == WheelNone {
		return false
	}

	wheelActive := false
	switch config.Wheel {
	case WheelUp:
		wheelActive = input.wheelDeltaY > 0
	case WheelDown:
		wheelActive = input.wheelDeltaY < 0
	case WheelLeft:
		wheelActive = input.wheelDeltaX < 0
	case WheelRight:
		wheelActive = input.wheelDeltaX > 0
	}

	return wheelActive &&
		self.checkModifiers(config.Modifiers) &&
		self.checkMouseModifiers(config.MouseButtonModifiers)
}

func (self *InputSystem) updateActionStates(input *InputComponent) {
	for action := range input.bindings {
		current := input.actionState[action]
		previous := input.previousState[action]
		input.justPressed[action] = current && !previous
		input.justReleased[action] = !current && previous
	}
}
