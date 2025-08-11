package input

import dinput "github.com/quasilyte/ebitengine-input"

var (
	system  dinput.System
	handler map[uint8]*dinput.Handler = make(map[uint8]*dinput.Handler)
)

func Initialize(id uint8, keymaps dinput.Keymap) {
	system.Init(dinput.SystemConfig{
		DevicesEnabled: dinput.AnyDevice,
	})
	handler[id] = system.NewHandler(id, keymaps)
}

func Update() {
	system.Update()
}

func ActionIsJustPressed(id uint8, action dinput.Action) bool {
	return handler[id].ActionIsJustPressed(action)
}

func ActionIsJustReleased(id uint8, action dinput.Action) bool {
	return handler[id].ActionIsJustReleased(action)
}

func ActionIsPressed(id uint8, action dinput.Action) bool {
	return handler[id].ActionIsPressed(action)
}
