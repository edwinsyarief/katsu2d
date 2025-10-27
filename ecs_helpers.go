package katsu2d

import (
	"image"

	"github.com/edwinsyarief/katsu2d/event"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/mlange-42/ark/ecs"
)

func updateHiResDisplayResource(w *ecs.World, width, height int) {
	display := ecs.GetResource[HiResDisplaySize](w)
	if display == nil {
		ecs.AddResource(w, &HiResDisplaySize{
			Width:  width,
			Height: height,
		})
	} else {
		display.Width = width
		display.Height = height
	}
}

func initializeAssetManagers(w *ecs.World,
	tm *TextureManager, fm *FontManager, am *AudioManager, shm *ShaderManager, scm *SceneManager) {
	if tmr := ecs.GetResource[TextureManager](w); tmr == nil {
		ecs.AddResource(w, tm)
	}
	if fmr := ecs.GetResource[FontManager](w); fmr == nil {
		ecs.AddResource(w, fm)
	}
	if amr := ecs.GetResource[AudioManager](w); amr == nil {
		ecs.AddResource(w, am)
	}
	if shrm := ecs.GetResource[ShaderManager](w); shrm == nil {
		ecs.AddResource(w, shm)
	}
	if scmr := ecs.GetResource[SceneManager](w); scmr == nil {
		ecs.AddResource(w, scm)
	}
}

func GetHiResDisplayInfo(w *ecs.World) *HiResDisplaySize {
	return ecs.GetResource[HiResDisplaySize](w)
}

func GetTextureManager(w *ecs.World) *TextureManager {
	return ecs.GetResource[TextureManager](w)
}

func GetFontManager(w *ecs.World) *FontManager {
	return ecs.GetResource[FontManager](w)
}

func GetAudioManager(w *ecs.World) *AudioManager {
	return ecs.GetResource[AudioManager](w)
}

func GetShaderManager(w *ecs.World) *ShaderManager {
	return ecs.GetResource[ShaderManager](w)
}

func GetSceneManager(w *ecs.World) *SceneManager {
	return ecs.GetResource[SceneManager](w)
}

func getEventBus(w *ecs.World) *event.EventBus {
	if eb := ecs.GetResource[event.EventBus](w); eb == nil {
		ecs.AddResource(w, &event.EventBus{})
	}
	return ecs.GetResource[event.EventBus](w)
}

func Subscribe[T any](w *ecs.World, fn func(T)) {
	eb := getEventBus(w)
	event.Subscribe(eb, fn)
}

func Publish[T any](w *ecs.World, o T) {
	eb := getEventBus(w)
	event.Publish(eb, o)
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
func Bind(input *InputComponent, action Action, primary InputCode, modifiers ...InputCode) {
	primaryCode := toInputCode(primary)
	mods := make([]InputCode, len(modifiers))
	for i, m := range modifiers {
		mods[i] = toInputCode(m)
	}
	binding := KeyConfig{
		Primary:   primaryCode,
		Modifiers: mods,
	}
	input.Bindings[action] = append(input.Bindings[action], binding)
}

// BatchBind replaces all existing bindings with a new set.
// This is useful for loading a saved or preset control scheme.
func BatchBind(input *InputComponent, bindings map[Action][]KeyConfig) {
	input.Bindings = make(map[Action][]KeyConfig) // Clear all existing bindings
	for a, b := range bindings {
		for _, k := range b {
			Bind(input, a, k.Primary, k.Modifiers...)
		}
	}
}

func NewKeyConfig(key any, modifiers ...any) KeyConfig {
	res := KeyConfig{
		Primary:   toInputCode(key),
		Modifiers: make([]InputCode, 0),
	}

	if len(modifiers) > 0 {
		for _, m := range modifiers {
			res.Modifiers = append(res.Modifiers, toInputCode(m))
		}
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
