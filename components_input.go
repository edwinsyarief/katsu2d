// input component
package katsu2d

import "github.com/hajimehoshi/ebiten/v2"

type Action string

type WheelDirection int

const (
	WheelNone WheelDirection = iota
	WheelUp
	WheelDown
	WheelLeft
	WheelRight
)

type KeyConfig struct {
	Key                      ebiten.Key
	Modifiers                []ebiten.Key
	GamepadButton            ebiten.GamepadButton
	GamepadModifiers         []ebiten.GamepadButton
	StandardGamepadButton    ebiten.StandardGamepadButton
	StandardGamepadModifiers []ebiten.StandardGamepadButton
	StandardGamepadAxis      ebiten.StandardGamepadAxis
	AxisThreshold            float64
	AxisDirection            int
	MouseButton              ebiten.MouseButton
	MouseButtonModifiers     []ebiten.MouseButton
	Wheel                    WheelDirection
}

type InputComponent struct {
	bindings      map[Action][]KeyConfig
	actionState   map[Action]bool
	justPressed   map[Action]bool
	justReleased  map[Action]bool
	previousState map[Action]bool
	pressDuration map[Action]int
	wheelState    map[Action]bool
	wheelDeltaX   float64
	wheelDeltaY   float64
}

func NewInputComponent(bindings map[Action][]KeyConfig) *InputComponent {
	ic := &InputComponent{
		bindings:      bindings,
		actionState:   make(map[Action]bool),
		justPressed:   make(map[Action]bool),
		justReleased:  make(map[Action]bool),
		previousState: make(map[Action]bool),
		pressDuration: make(map[Action]int),
		wheelState:    make(map[Action]bool),
	}
	for action := range bindings {
		ic.actionState[action] = false
		ic.justPressed[action] = false
		ic.justReleased[action] = false
		ic.previousState[action] = false
		ic.pressDuration[action] = 0
		ic.wheelState[action] = false
	}
	return ic
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

func (self *InputComponent) GetWheelDeltaX() float64 {
	return self.wheelDeltaX
}

func (self *InputComponent) GetWheelDeltaY() float64 {
	return self.wheelDeltaY
}

func (self *InputComponent) IsWheel(action Action) bool {
	return self.wheelState[action]
}

func (self *InputComponent) GetPressDuration(action Action) int {
	return self.pressDuration[action]
}
