package katsu2d

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Action represents a game action (e.g., "move_up", "jump").
type Action string

// InputType distinguishes between different types of input devices.
type InputType int

const (
	InputTypeKeyboard InputType = 0
	InputTypeMouse    InputType = 1
	InputTypeGamepad  InputType = 2
	InputTypeAnalog   InputType = 3 // New type for analog sticks
)

// InputCode represents a specific key, button, or axis on a device.
type InputCode struct {
	Type InputType
	Code int
}

// Binding holds a primary input and optional modifiers for an action.
type Binding struct {
	Primary   InputCode
	Modifiers []InputCode
}

// KeyConfig is a helper struct for defining bindings in a more readable way.
type KeyConfig struct {
	Primary   any
	Modifiers []any
}

// InputComponent stores all input bindings and their current state.
type InputComponent struct {
	Bindings     map[Action][]Binding
	JustPressed  map[Action]bool
	Pressed      map[Action]bool
	JustReleased map[Action]bool

	// Mouse wheel state, stored separately as it's not a binary button state
	MouseWheelX float64
	MouseWheelY float64
}

func (self *InputComponent) Init(bindinds map[Action][]KeyConfig) {
	self.Bindings = make(map[Action][]Binding)
	self.JustPressed = make(map[Action]bool)
	self.Pressed = make(map[Action]bool)
	self.JustReleased = make(map[Action]bool)

	if len(bindinds) > 0 {
		self.BatchBind(bindinds)
	}
}

// toInputCode converts a generic Ebitengine input type to a standardized InputCode.
func toInputCode(v any) InputCode {
	switch code := v.(type) {
	case ebiten.Key:
		return InputCode{Type: InputTypeKeyboard, Code: int(code)}
	case ebiten.MouseButton:
		return InputCode{Type: InputTypeMouse, Code: int(code)}
	case ebiten.GamepadButton:
		return InputCode{Type: InputTypeGamepad, Code: int(code)}
	case ebiten.StandardGamepadButton:
		return InputCode{Type: InputTypeGamepad, Code: int(code)}
	case ebiten.StandardGamepadAxis:
		return InputCode{Type: InputTypeAnalog, Code: int(code)}
	default:
		// Panicking here is okay for development, but in a production game
		// this should be handled gracefully with an error log.
		panic("unsupported input type")
	}
}

// Bind adds a new binding for a specific action.
func (self *InputComponent) Bind(action Action, primary any, modifiers ...any) {
	primaryCode := toInputCode(primary)
	mods := make([]InputCode, len(modifiers))
	for i, m := range modifiers {
		mods[i] = toInputCode(m)
	}
	binding := Binding{
		Primary:   primaryCode,
		Modifiers: mods,
	}
	self.Bindings[action] = append(self.Bindings[action], binding)
}

// BatchBind replaces all existing bindings with a new set.
// This is useful for loading a saved or preset control scheme.
func (self *InputComponent) BatchBind(bindings map[Action][]KeyConfig) {
	self.Bindings = make(map[Action][]Binding) // Clear all existing bindings
	for a, b := range bindings {
		for _, k := range b {
			self.Bind(a, k.Primary, k.Modifiers...)
		}
	}
}

// IsPressed returns true if the action is currently being pressed.
func (self *InputComponent) IsPressed(action Action) bool {
	return self.Pressed[action]
}

// IsJustPressed returns true if the action was just pressed in the current frame.
func (self *InputComponent) IsJustPressed(action Action) bool {
	return self.JustPressed[action]
}

// IsJustReleased returns true if the action was just released in the current frame.
func (self *InputComponent) IsJustReleased(action Action) bool {
	return self.JustReleased[action]
}

// GetPressDuration returns the duration (in seconds) the action has been held down.
func (self *InputComponent) GetPressDuration(action Action) float64 {
	var duration float64
	if !self.Pressed[action] {
		return 0.0
	}
	// An action can have multiple bindings. We want the longest duration.
	for _, binding := range self.Bindings[action] {
		var currentDuration float64
		switch binding.Primary.Type {
		case InputTypeKeyboard:
			currentDuration = float64(inpututil.KeyPressDuration(ebiten.Key(binding.Primary.Code)))
		case InputTypeMouse:
			currentDuration = float64(inpututil.MouseButtonPressDuration(ebiten.MouseButton(binding.Primary.Code)))
		case InputTypeGamepad:
			currentDuration = float64(inpututil.GamepadButtonPressDuration(0, ebiten.GamepadButton(binding.Primary.Code)))
			// TODO: need to think how to optimize this
			if currentDuration == 0 {
				currentDuration = float64(inpututil.StandardGamepadButtonPressDuration(0, ebiten.StandardGamepadButton(binding.Primary.Code)))
			}
		}
		if currentDuration > duration {
			duration = currentDuration
		}
	}
	return duration / float64(ebiten.TPS())
}

func (self *InputComponent) GetCustomGamepadPressDuration(id ebiten.GamepadID, code ebiten.GamepadButton) float64 {
	duration := float64(inpututil.GamepadButtonPressDuration(id, code))
	return duration / float64(ebiten.TPS())
}

func (self *InputComponent) GetCustomStandardGamepadPressDuration(id ebiten.GamepadID, code ebiten.StandardGamepadButton) float64 {
	duration := float64(inpututil.StandardGamepadButtonPressDuration(id, code))
	return duration / float64(ebiten.TPS())
}

// GetStandardGamepadAxis returns the current value of a given gamepad axis.
func (self *InputComponent) GetStandardGamepadAxis(id ebiten.GamepadID, axis ebiten.StandardGamepadAxis) float64 {
	return ebiten.StandardGamepadAxisValue(id, axis)
}

func (self *InputComponent) GetGamepadAxis(id ebiten.GamepadID, axis ebiten.GamepadAxisType) float64 {
	return ebiten.GamepadAxisValue(id, axis)
}

func (self *InputComponent) IsCustomGamepadButtonPressed(id ebiten.GamepadID, code any) bool {
	return ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(code.(int))) ||
		ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButton(code.(int)))
}

func (self *InputComponent) IsCustomGamepadButtonJustPressed(id ebiten.GamepadID, code any) bool {
	return inpututil.IsGamepadButtonJustPressed(id, ebiten.GamepadButton(code.(int))) ||
		inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButton(code.(int)))
}

func (self *InputComponent) IsCustomGamepadButtonJustReleased(id ebiten.GamepadID, code any) bool {
	return inpututil.IsGamepadButtonJustReleased(id, ebiten.GamepadButton(code.(int))) ||
		inpututil.IsStandardGamepadButtonJustReleased(id, ebiten.StandardGamepadButton(code.(int)))
}

// isPressed checks if the given InputCode is currently pressed.
func isPressed(code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return ebiten.IsMouseButtonPressed(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return ebiten.IsKeyPressed(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return ebiten.IsGamepadButtonPressed(ebiten.GamepadID(0), ebiten.GamepadButton(code.Code)) ||
			ebiten.IsStandardGamepadButtonPressed(ebiten.GamepadID(0), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}

// isJustPressed checks if the given InputCode was pressed in the current frame.
func isJustPressed(code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return inpututil.IsKeyJustPressed(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return inpututil.IsGamepadButtonJustPressed(ebiten.GamepadID(0), ebiten.GamepadButton(code.Code)) ||
			inpututil.IsStandardGamepadButtonJustPressed(ebiten.GamepadID(0), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}

// isJustReleased checks if the given InputCode was released in the current frame.
func isJustReleased(code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return inpututil.IsKeyJustReleased(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return inpututil.IsGamepadButtonJustReleased(ebiten.GamepadID(0), ebiten.GamepadButton(code.Code)) ||
			inpututil.IsStandardGamepadButtonJustReleased(ebiten.GamepadID(0), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}
