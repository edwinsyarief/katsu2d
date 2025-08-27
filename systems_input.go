package katsu2d

// InputSystem is responsible for updating the state of all InputComponents.
type InputSystem struct{}

// NewInputSystem creates and initializes an InputSystem.
func NewInputSystem() *InputSystem {
	return &InputSystem{}
}

// Update processes all input components in the world.
func (s *InputSystem) Update(world *World, delta float64) {
	// First, reset all input states from the previous frame.
	entities := world.Query(CTInput)
	for _, e := range entities {
		compAny, _ := world.GetComponent(e, CTInput)
		comp := compAny.(*InputComponent)
		for action := range comp.bindings {
			comp.justPressed[action] = false
			comp.pressed[action] = false
			comp.JustReleased[action] = false
		}
		// Now, check for the current state for all bindings.
		for action := range comp.bindings {
			for _, binding := range comp.bindings[action] {
				modsDown := true
				for _, mod := range binding.Modifiers {
					if !isPressed(mod) {
						modsDown = false
						break
					}
				}

				if isJustPressed(binding.Primary) {
					comp.justPressed[action] = modsDown
				}
				if isPressed(binding.Primary) {
					comp.pressed[action] = modsDown
				}
				if isJustReleased(binding.Primary) {
					comp.JustReleased[action] = modsDown
				}
			}
		}
	}
}
