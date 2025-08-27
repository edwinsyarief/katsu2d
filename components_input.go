package katsu2d

import "github.com/hajimehoshi/ebiten/v2"

type Action string

type KeyConfig struct {
	Key                  ebiten.Key
	Modifiers            []ebiten.Key
	GamepadButton        ebiten.GamepadButton
	GamepadModifiers     []ebiten.GamepadButton
	MouseButton          ebiten.MouseButton
	MouseButtonModifiers []ebiten.MouseButton
}

type InputComponent struct {
	bindings      map[Action][]KeyConfig
	actionState   map[Action]bool
	justPressed   map[Action]bool
	justReleased  map[Action]bool
	previousState map[Action]bool
}

func NewInputComponent(bindings map[Action][]KeyConfig) *InputComponent {
	return &InputComponent{
		bindings:      bindings,
		actionState:   make(map[Action]bool),
		justPressed:   make(map[Action]bool),
		justReleased:  make(map[Action]bool),
		previousState: make(map[Action]bool),
	}
}

func (self *InputComponent) IsPressed(action Action) bool {
	return self.actionState[action]
}

func (self *InputComponent) IsJustPressed(action Action) bool {
	return self.justPressed[action]
}

func (self *InputComponent) IsJustReleased(action Action) bool {
	return self.justReleased[action]
}
