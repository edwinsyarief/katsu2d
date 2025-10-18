package katsu2d

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

// KeyConfig is a helper struct for defining bindings in a more readable way.
type KeyConfig struct {
	Primary   InputCode
	Modifiers []InputCode
}

// InputComponent stores all input bindings and their current state.
type InputComponent struct {
	ID           int
	Bindings     map[Action][]KeyConfig
	JustPressed  map[Action]bool
	Pressed      map[Action]bool
	JustReleased map[Action]bool

	// Mouse wheel state, stored separately as it's not a binary button state
	MouseWheelX float64
	MouseWheelY float64
}
