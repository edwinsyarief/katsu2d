package katsu2d

import (
	"image"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func updateHiResDisplayResource(w *teishoku.World, width, height int) {
	if ok, _ := teishoku.HasResource[HiResDisplaySize](w.Resources()); !ok {
		w.Resources().Add(&HiResDisplaySize{
			Width:  width,
			Height: height,
		})
	} else {
		layout, _ := teishoku.GetResource[HiResDisplaySize](w.Resources())
		layout.Width = width
		layout.Height = height
	}
}

func initializeAssetManagers(w *teishoku.World,
	tm *TextureManager, fm *FontManager, am *AudioManager, shm *ShaderManager, scm *SceneManager) {
	if ok, _ := teishoku.HasResource[TextureManager](w.Resources()); !ok {
		w.Resources().Add(tm)
	}
	if ok, _ := teishoku.HasResource[FontManager](w.Resources()); !ok {
		w.Resources().Add(fm)
	}
	if ok, _ := teishoku.HasResource[AudioManager](w.Resources()); !ok {
		w.Resources().Add(am)
	}
	if ok, _ := teishoku.HasResource[ShaderManager](w.Resources()); !ok {
		w.Resources().Add(shm)
	}
	if ok, _ := teishoku.HasResource[SceneManager](w.Resources()); !ok {
		w.Resources().Add(scm)
	}
}

func GetTextureManager(w *teishoku.World) *TextureManager {
	res, _ := teishoku.GetResource[TextureManager](w.Resources())
	return res
}

func GetFontManager(w *teishoku.World) *FontManager {
	res, _ := teishoku.GetResource[FontManager](w.Resources())
	return res
}

func GetAudioManager(w *teishoku.World) *AudioManager {
	res, _ := teishoku.GetResource[AudioManager](w.Resources())
	return res
}

func GetShaderManager(w *teishoku.World) *ShaderManager {
	res, _ := teishoku.GetResource[ShaderManager](w.Resources())
	return res
}

func GetSceneManager(w *teishoku.World) *SceneManager {
	res, _ := teishoku.GetResource[SceneManager](w.Resources())
	return res
}

func getEventBus(w *teishoku.World) *teishoku.EventBus {
	if ok, _ := teishoku.HasResource[teishoku.EventBus](w.Resources()); !ok {
		w.Resources().Add(&teishoku.EventBus{})
	}
	eb, _ := teishoku.GetResource[teishoku.EventBus](w.Resources())
	return eb
}

func Subscribe[T any](w *teishoku.World, fn func(T)) {
	eb := getEventBus(w)
	teishoku.Subscribe(eb, fn)
}

func Publish[T any](w *teishoku.World, o T) {
	eb := getEventBus(w)
	teishoku.Publish(eb, o)
}

func (self *Transform) SetFromComponent(comp *TransformComponent) {
	self.Reset()
	self.SetPosition(Vector(comp.Position))
	self.SetOffset(Vector(comp.Offset))
	self.SetOrigin(Vector(comp.Origin))
	self.SetScale(Vector(comp.Scale))
	self.SetRotation(comp.Rotation)
}

func GenerateMesh(m *MeshComponent, s *SpriteComponent) {
	if m.IsDirty || m.MeshType != MeshTypeCustom {
		var srcRect image.Rectangle
		if IsBoundEmpty(s.Bound) {
			srcRect = image.Rect(0, 0, s.Width, s.Height)
		} else {
			srcRect = image.Rect(int(s.Bound.Min.X), int(s.Bound.Min.Y), s.Width, s.Height)
		}

		numVertices := (m.Rows + 1) * (m.Cols + 1)
		m.BaseVertices = make([]ebiten.Vertex, numVertices)
		m.Vertices = make([]ebiten.Vertex, numVertices)

		for r := 0; r <= m.Rows; r++ {
			for c := 0; c <= m.Cols; c++ {
				idx := r*(m.Cols+1) + c
				vx := float32(float32(s.Width) * (float32(c) / float32(m.Cols)))
				vy := float32(float32(s.Height) * (float32(r) / float32(m.Rows)))
				u := float32(srcRect.Min.X) + float32(srcRect.Dx())*(float32(c)/float32(m.Cols))
				v := float32(srcRect.Min.Y) + float32(srcRect.Dy())*(float32(r)/float32(m.Rows))

				m.Vertices[idx] = ebiten.Vertex{
					DstX:   vx,
					DstY:   vy,
					SrcX:   u,
					SrcY:   v,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				}
			}
		}

		numQuads := m.Rows * m.Cols
		m.Indices = make([]uint16, numQuads*6)
		i := 0
		for r := 0; r < m.Rows; r++ {
			for c := 0; c < m.Cols; c++ {
				topLeft := uint16(r*(m.Cols+1) + c)
				topRight := topLeft + 1
				bottomLeft := uint16((r+1)*(m.Cols+1) + c)
				bottomRight := bottomLeft + 1
				m.Indices[i] = topLeft
				m.Indices[i+1] = topRight
				m.Indices[i+2] = bottomLeft
				m.Indices[i+3] = bottomLeft
				m.Indices[i+4] = topRight
				m.Indices[i+5] = bottomRight
				i += 6
			}
		}

		m.BaseVertices = make([]ebiten.Vertex, len(m.Vertices))
		copy(m.BaseVertices, m.Vertices)

		m.IsDirty = false
	}
}

// Bind adds a new binding for a specific action.
func Bind(input *InputComponent, action Action, primary any, modifiers ...any) {
	primaryCode := toInputCode(primary)
	mods := make([]InputCode, len(modifiers))
	for i, m := range modifiers {
		mods[i] = toInputCode(m)
	}
	binding := Binding{
		Primary:   primaryCode,
		Modifiers: mods,
	}
	input.Bindings[action] = append(input.Bindings[action], binding)
}

// BatchBind replaces all existing bindings with a new set.
// This is useful for loading a saved or preset control scheme.
func BatchBind(input *InputComponent, bindings map[Action][]KeyConfig) {
	input.Bindings = make(map[Action][]Binding) // Clear all existing bindings
	for a, b := range bindings {
		for _, k := range b {
			Bind(input, a, k.Primary, k.Modifiers...)
		}
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

// isPressed checks if the given InputCode is currently pressed.
func isPressed(id int, code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return ebiten.IsMouseButtonPressed(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return ebiten.IsKeyPressed(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return ebiten.IsGamepadButtonPressed(ebiten.GamepadID(id), ebiten.GamepadButton(code.Code)) ||
			ebiten.IsStandardGamepadButtonPressed(ebiten.GamepadID(id), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}

// isJustPressed checks if the given InputCode was pressed in the current frame.
func isJustPressed(id int, code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return inpututil.IsMouseButtonJustPressed(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return inpututil.IsKeyJustPressed(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return inpututil.IsGamepadButtonJustPressed(ebiten.GamepadID(id), ebiten.GamepadButton(code.Code)) ||
			inpututil.IsStandardGamepadButtonJustPressed(ebiten.GamepadID(id), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}

// isJustReleased checks if the given InputCode was released in the current frame.
func isJustReleased(id int, code InputCode) bool {
	switch code.Type {
	case InputTypeMouse:
		return inpututil.IsMouseButtonJustReleased(ebiten.MouseButton(code.Code))
	case InputTypeKeyboard:
		return inpututil.IsKeyJustReleased(ebiten.Key(code.Code))
	case InputTypeGamepad:
		return inpututil.IsGamepadButtonJustReleased(ebiten.GamepadID(id), ebiten.GamepadButton(code.Code)) ||
			inpututil.IsStandardGamepadButtonJustReleased(ebiten.GamepadID(id), ebiten.StandardGamepadButton(code.Code))
	}
	return false
}
