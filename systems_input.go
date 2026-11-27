package katsu2d

import (
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

// InputSystem polls device input states for all InputComponent entities.
type InputSystem struct {
	filter      *teishoku.Filter[InputComponent]
	blockHandle teishoku.ResourceHandle[inputBlockState]
	initialized bool
}

// NewInputSystem creates a new InputSystem.
func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

// Initialize prepares the InputSystem's ECS filter.
func (s *InputSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter[InputComponent](w)
	res := w.Resources()
	if ok, _ := teishoku.HasResource[inputBlockState](res); !ok {
		teishoku.AddResource(res, &inputBlockState{})
	}
	s.blockHandle = teishoku.UseResource[inputBlockState](res)
	s.initialized = true
}

// Update polls keyboard, mouse, and gamepad states for all InputComponent entities.
func (s *InputSystem) Update(w *teishoku.World, dt float64) {
	if !s.initialized {
		s.Initialize(w)
	}

	if state := s.blockHandle.Get(); state != nil && state.blocked {
		s.filter.EachChunk(func(inputs []InputComponent) {
			for i := range inputs {
				inp := &inputs[i]
				ensureInputStateMaps(inp)
				for action := range inp.JustPressed {
					inp.JustPressed[action] = false
				}
				for action := range inp.Pressed {
					inp.Pressed[action] = false
				}
				for action := range inp.JustReleased {
					inp.JustReleased[action] = false
				}
				inp.MouseWheelX = 0
				inp.MouseWheelY = 0
			}
		})
		return
	}

	wx, wy := ebiten.Wheel()

	s.filter.EachChunk(func(inputs []InputComponent) {
		for i := range inputs {
			inp := &inputs[i]
			ensureInputStateMaps(inp)

			// Reset states for the current frame.
			for action := range inp.Bindings {
				inp.JustPressed[action] = false
				inp.JustReleased[action] = false
			}

			inp.MouseWheelX = wx
			inp.MouseWheelY = wy

			for action, bindings := range inp.Bindings {
				isAnyJustPressed := false
				isAnyPressed := false
				isAnyJustReleased := false

				for j := range bindings {
					binding := &bindings[j]
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

				inp.JustPressed[action] = isAnyJustPressed
				inp.JustReleased[action] = isAnyJustReleased
				inp.Pressed[action] = isAnyPressed
			}
		}
	})
}

func ensureInputStateMaps(inp *InputComponent) {
	if inp.JustPressed == nil {
		inp.JustPressed = make(map[Action]bool, len(inp.Bindings))
	}
	if inp.Pressed == nil {
		inp.Pressed = make(map[Action]bool, len(inp.Bindings))
	}
	if inp.JustReleased == nil {
		inp.JustReleased = make(map[Action]bool, len(inp.Bindings))
	}
}
