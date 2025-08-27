package katsu2d

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	InvalidKey            = ebiten.Key(-1)
	InvalidMouse          = ebiten.MouseButton(-1)
	InvalidGamepad        = ebiten.GamepadButton(-1)
	InvalidStdButton      = ebiten.StandardGamepadButton(-1)
	InvalidStdAxis        = ebiten.StandardGamepadAxis(-1)
	GamepadUpdateInterval = 60 // Update gamepad IDs every 60 frames
)

// InputSystem is an UpdateSystem that handles all game input.
type InputSystem struct {
	gamepadIDs []ebiten.GamepadID
}

// NewInputSystem creates a new input system
func NewInputSystem() *InputSystem {
	return &InputSystem{
		gamepadIDs: ebiten.AppendGamepadIDs(nil),
	}
}

// Update implements the UpdateSystem interface
func (self *InputSystem) Update(world *World, _ float64) {
	// Update gamepad IDs periodically
	if uint(ebiten.ActualTPS())%GamepadUpdateInterval == 0 {
		self.gamepadIDs = ebiten.AppendGamepadIDs(nil)
	}

	entities := world.Query(CTInput)
	for _, e := range entities {
		comp, _ := world.GetComponent(e, CTInput)
		inputComp := comp.(*InputComponent)
		self.updateInputComponent(inputComp)
	}
}

func (self *InputSystem) updateInputComponent(inputComp *InputComponent) {
	// Copy current state to previous state
	for action, isPressed := range inputComp.actionState {
		inputComp.previousState[action] = isPressed
	}

	// Reset states
	self.resetStates(inputComp)

	// Get wheel delta
	inputComp.wheelDeltaX, inputComp.wheelDeltaY = ebiten.Wheel()

	// Evaluate actions
	for action, configs := range inputComp.bindings {
		self.evaluateAction(action, configs, inputComp)
	}
}

func (self *InputSystem) resetStates(inputComp *InputComponent) {
	for action := range inputComp.bindings {
		inputComp.actionState[action] = false
		inputComp.wheelState[action] = false
		inputComp.pressDuration[action] = 0
	}
}

func (self *InputSystem) evaluateAction(action Action, configs []KeyConfig, inputComp *InputComponent) {
	actionIsPressed := false
	maxDuration := 0

	for _, config := range configs {
		pressed, duration := self.evaluateInputConfig(config, inputComp)
		if pressed {
			actionIsPressed = true
			if duration > maxDuration {
				maxDuration = duration
			}
		}
	}

	inputComp.actionState[action] = actionIsPressed
	if actionIsPressed {
		inputComp.pressDuration[action] = maxDuration
	}

	inputComp.justPressed[action] = actionIsPressed && !inputComp.previousState[action]
	inputComp.justReleased[action] = !actionIsPressed && inputComp.previousState[action]
}

func (self *InputSystem) evaluateInputConfig(config KeyConfig, inputComp *InputComponent) (bool, int) {
	switch {
	case config.Key != InvalidKey:
		return self.checkKeyboard(config)
	case config.MouseButton != InvalidMouse:
		return self.checkMouseButton(config)
	case config.Wheel != WheelNone:
		return self.checkWheel(config, inputComp)
	case config.GamepadButton != InvalidGamepad:
		return self.checkGamepadButton(config)
	case config.StandardGamepadButton != InvalidStdButton:
		return self.checkStandardGamepadButton(config)
	case config.StandardGamepadAxis != InvalidStdAxis:
		return self.checkStandardGamepadAxis(config)
	}
	return false, 0
}

func (self *InputSystem) checkKeyboard(config KeyConfig) (bool, int) {
	if ebiten.IsKeyPressed(config.Key) {
		if checkKeyboardModifiers(config) && checkMouseButtonModifiers(config) {
			return true, inpututil.KeyPressDuration(config.Key)
		}
	}
	return false, 0
}

func (self *InputSystem) checkMouseButton(config KeyConfig) (bool, int) {
	if ebiten.IsMouseButtonPressed(config.MouseButton) {
		if checkKeyboardModifiers(config) && checkMouseButtonModifiers(config) {
			return true, inpututil.MouseButtonPressDuration(config.MouseButton)
		}
	}
	return false, 0
}

func (self *InputSystem) checkWheel(config KeyConfig, inputComp *InputComponent) (bool, int) {
	if isWheelActive(config, inputComp) {
		if checkKeyboardModifiers(config) && checkMouseButtonModifiers(config) {
			return true, 0
		}
	}
	return false, 0
}

func (self *InputSystem) checkGamepadButton(config KeyConfig) (bool, int) {
	for _, gID := range self.gamepadIDs {
		if ebiten.IsGamepadButtonPressed(gID, config.GamepadButton) {
			if checkGamepadModifiers(gID, config) {
				return true, inpututil.GamepadButtonPressDuration(gID, config.GamepadButton)
			}
		}
	}
	return false, 0
}

func (self *InputSystem) checkStandardGamepadButton(config KeyConfig) (bool, int) {
	for _, gID := range self.gamepadIDs {
		if ebiten.IsStandardGamepadButtonPressed(gID, config.StandardGamepadButton) {
			if checkStandardGamepadModifiers(gID, config) {
				return true, inpututil.StandardGamepadButtonPressDuration(gID, config.StandardGamepadButton)
			}
		}
	}
	return false, 0
}

func (self *InputSystem) checkStandardGamepadAxis(config KeyConfig) (bool, int) {
	for _, gID := range self.gamepadIDs {
		axisValue := ebiten.StandardGamepadAxisValue(gID, config.StandardGamepadAxis)
		if math.Abs(axisValue) > config.AxisThreshold && (axisValue*float64(config.AxisDirection) > 0) {
			if checkStandardGamepadModifiers(gID, config) {
				return true, 0
			}
		}
	}
	return false, 0
}

// Helper functions (unchanged from original)
func checkKeyboardModifiers(config KeyConfig) bool {
	for _, mod := range config.Modifiers {
		if !ebiten.IsKeyPressed(mod) {
			return false
		}
	}
	return true
}

func checkMouseButtonModifiers(config KeyConfig) bool {
	for _, mod := range config.MouseButtonModifiers {
		if !ebiten.IsMouseButtonPressed(mod) {
			return false
		}
	}
	return true
}

func checkGamepadModifiers(gamepadID ebiten.GamepadID, config KeyConfig) bool {
	for _, mod := range config.GamepadModifiers {
		if !ebiten.IsGamepadButtonPressed(gamepadID, mod) {
			return false
		}
	}
	return true
}

func checkStandardGamepadModifiers(gamepadID ebiten.GamepadID, config KeyConfig) bool {
	for _, mod := range config.StandardGamepadModifiers {
		if !ebiten.IsStandardGamepadButtonPressed(gamepadID, mod) {
			return false
		}
	}
	return true
}

func isWheelActive(config KeyConfig, inputComp *InputComponent) bool {
	return (config.Wheel == WheelUp && inputComp.wheelDeltaY > 0) ||
		(config.Wheel == WheelDown && inputComp.wheelDeltaY < 0) ||
		(config.Wheel == WheelLeft && inputComp.wheelDeltaX < 0) ||
		(config.Wheel == WheelRight && inputComp.wheelDeltaX > 0)
}
