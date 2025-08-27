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

type KeyConfig struct {
	Primary   any
	Modifiers []any
}

// InputComponent stores all input bindings and their current state.
type InputComponent struct {
	bindings     map[Action][]Binding
	justPressed  map[Action]bool
	pressed      map[Action]bool
	JustReleased map[Action]bool
}

// NewInputComponent creates and initializes a new InputComponent.
func NewInputComponent(bindinds map[Action][]KeyConfig) *InputComponent {
	res := &InputComponent{
		bindings:     make(map[Action][]Binding),
		justPressed:  make(map[Action]bool),
		pressed:      make(map[Action]bool),
		JustReleased: make(map[Action]bool),
	}

	if len(bindinds) > 0 {
		res.BatchBind(bindinds)
	}

	return res
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
	default:
		// This should be improved with more graceful error handling in a real game.
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
	self.bindings[action] = append(self.bindings[action], binding)
}

// BatchBind replaces all existing bindings with a new set.
// This is useful for loading a saved or preset control scheme.
func (self *InputComponent) BatchBind(bindings map[Action][]KeyConfig) {
	for a, b := range bindings {
		// Clear existing bindings for this action to avoid duplicates
		self.bindings[a] = []Binding{}
		for _, k := range b {
			self.Bind(a, k.Primary, k.Modifiers...)
		}
	}
}

func (self *InputComponent) IsPressed(action Action) bool {
	return self.pressed[action]
}

func (self *InputComponent) IsJustPressed(action Action) bool {
	return self.justPressed[action]
}

func (self *InputComponent) IsJustReleased(action Action) bool {
	return self.JustReleased[action]
}

// isPressed checks if the given InputCode is currently pressed.
func isPressed(code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return ebiten.IsMouseButtonPressed(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return ebiten.IsKeyPressed(ebiten.Key(code.Code))
	case InputTypeGamepad:
		// Note: This only checks for gamepad 0. You might want to extend this
		// to handle multiple gamepads.
		return ebiten.IsGamepadButtonPressed(0, ebiten.GamepadButton(code.Code))
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
		return inpututil.IsGamepadButtonJustPressed(0, ebiten.GamepadButton(code.Code))
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
		return inpututil.IsGamepadButtonJustReleased(0, ebiten.GamepadButton(code.Code))
	}
	return false
}
