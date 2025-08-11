package input

import dinput "github.com/quasilyte/ebitengine-input"

func MergeKeymaps(keymaps ...dinput.Keymap) dinput.Keymap {
	return dinput.MergeKeymaps(keymaps...)
}
