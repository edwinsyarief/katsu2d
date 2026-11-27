package katsu2d

import (
	"image"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func updateHiResDisplayResource(w *teishoku.World, width, height int) {
	if ok, _ := teishoku.HasResource[HiResDisplaySize](w.Resources()); !ok {
		teishoku.AddResource(w.Resources(), &HiResDisplaySize{
			Width:  width,
			Height: height,
		})
	} else {
		layout, _ := teishoku.GetResource[HiResDisplaySize](w.Resources())
		layout.Width = width
		layout.Height = height
	}
}

func initializeAssetManagers(
	w *teishoku.World,
	tm *TextureManager,
	fm *FontManager,
	am *AudioManager,
	shm *ShaderManager,
	scm *SceneManager,
) {
	res := w.Resources()
	if tm != nil {
		if ok, _ := teishoku.HasResource[TextureManager](res); !ok {
			teishoku.AddResource(res, tm)
		}
	}
	if fm != nil {
		if ok, _ := teishoku.HasResource[FontManager](res); !ok {
			teishoku.AddResource(res, fm)
		}
	}
	if am != nil {
		if ok, _ := teishoku.HasResource[AudioManager](res); !ok {
			teishoku.AddResource(res, am)
		}
	}
	if shm != nil {
		if ok, _ := teishoku.HasResource[ShaderManager](res); !ok {
			teishoku.AddResource(res, shm)
		}
	}
	if scm != nil {
		if ok, _ := teishoku.HasResource[SceneManager](res); !ok {
			teishoku.AddResource(res, scm)
		}
	}
}

// GetHiResDisplayInfo returns the current HiResDisplaySize resource from the world.
func GetHiResDisplayInfo(w *teishoku.World) *HiResDisplaySize {
	res, _ := teishoku.GetResource[HiResDisplaySize](w.Resources())
	return res
}

// GetTextureManager retrieves the TextureManager resource from the world.
func GetTextureManager(w *teishoku.World) *TextureManager {
	res, _ := teishoku.GetResource[TextureManager](w.Resources())
	return res
}

// GetFontManager retrieves the FontManager resource from the world.
func GetFontManager(w *teishoku.World) *FontManager {
	res, _ := teishoku.GetResource[FontManager](w.Resources())
	return res
}

// GetAudioManager retrieves the AudioManager resource from the world.
func GetAudioManager(w *teishoku.World) *AudioManager {
	res, _ := teishoku.GetResource[AudioManager](w.Resources())
	return res
}

// GetShaderManager retrieves the ShaderManager resource from the world.
func GetShaderManager(w *teishoku.World) *ShaderManager {
	res, _ := teishoku.GetResource[ShaderManager](w.Resources())
	return res
}

// GetSceneManager retrieves the SceneManager resource from the world.
func GetSceneManager(w *teishoku.World) *SceneManager {
	res, _ := teishoku.GetResource[SceneManager](w.Resources())
	return res
}

// GetActiveCamera retrieves the ActiveCamera resource from the world,
// returning nil if absent or no camera is currently active.
func GetActiveCamera(w *teishoku.World) *ActiveCamera {
	if w == nil {
		return nil
	}
	cam, _ := teishoku.GetResource[ActiveCamera](w.Resources())
	if cam == nil || !cam.HasCamera {
		return nil
	}
	return cam
}

// GetEventBus returns the world's EventBus resource, creating it lazily if absent.
func GetEventBus(w *teishoku.World) *teishoku.EventBus {
	return getEventBus(w)
}

func getEventBus(w *teishoku.World) *teishoku.EventBus {
	res := w.Resources()
	if ok, _ := teishoku.HasResource[teishoku.EventBus](res); !ok {
		teishoku.AddResource(res, teishoku.NewEventBus())
	}
	eb, _ := teishoku.GetResource[teishoku.EventBus](res)
	return eb
}

// Subscribe registers a typed event handler on the world's EventBus.
func Subscribe[T any](w *teishoku.World, fn func(T)) {
	teishoku.Subscribe(getEventBus(w), fn)
}

// Publish emits a typed event on the world's EventBus.
func Publish[T any](w *teishoku.World, event T) {
	teishoku.Publish(getEventBus(w), event)
}

// SetFromComponent copies all transform properties from a TransformComponent.
func (t *Transform) SetFromComponent(comp *TransformComponent) {
	t.Reset()
	t.SetPosition(Vector(comp.Position))
	t.SetOffset(Vector(comp.Offset))
	t.SetOrigin(Vector(comp.Origin))
	t.SetScale(Vector(comp.Scale))
	t.SetRotation(comp.Rotation)
}

// resolveSpriteSourceBounds resolves the source UV bounds (in img's coordinate space,
// including atlas sub-image offset) for sp. SpriteComponent.Bound is treated as
// texture-local coordinates and is never mutated.
func resolveSpriteSourceBounds(sp *SpriteComponent, img *ebiten.Image) (minX, minY, maxX, maxY float32) {
	if sp == nil {
		if img != nil {
			b := img.Bounds()
			return float32(b.Min.X), float32(b.Min.Y), float32(b.Max.X), float32(b.Max.Y)
		}
		return 0, 0, 0, 0
	}
	if img == nil {
		if !IsBoundEmpty(sp.Bound) {
			return float32(sp.Bound.Min.X), float32(sp.Bound.Min.Y), float32(sp.Bound.Max.X), float32(sp.Bound.Max.Y)
		}
		return 0, 0, float32(sp.Width), float32(sp.Height)
	}
	b := img.Bounds()
	if IsBoundEmpty(sp.Bound) {
		return float32(b.Min.X), float32(b.Min.Y), float32(b.Max.X), float32(b.Max.Y)
	}
	offX := float64(b.Min.X)
	offY := float64(b.Min.Y)
	return float32(sp.Bound.Min.X + offX),
		float32(sp.Bound.Min.Y + offY),
		float32(sp.Bound.Max.X + offX),
		float32(sp.Bound.Max.Y + offY)
}

// resolveSpriteSourceRect resolves the integer source rectangle (in img's coordinate
// space, including atlas sub-image offset) for sp without mutating sp.Bound.
func resolveSpriteSourceRect(sp *SpriteComponent, img *ebiten.Image) image.Rectangle {
	if sp == nil {
		if img != nil {
			return img.Bounds()
		}
		return image.Rectangle{}
	}
	if img == nil {
		if !IsBoundEmpty(sp.Bound) {
			return image.Rect(int(sp.Bound.Min.X), int(sp.Bound.Min.Y), int(sp.Bound.Max.X), int(sp.Bound.Max.Y))
		}
		return image.Rect(0, 0, sp.Width, sp.Height)
	}
	b := img.Bounds()
	if IsBoundEmpty(sp.Bound) {
		return b
	}
	return image.Rect(
		int(sp.Bound.Min.X)+b.Min.X,
		int(sp.Bound.Min.Y)+b.Min.Y,
		int(sp.Bound.Max.X)+b.Min.X,
		int(sp.Bound.Max.Y)+b.Min.Y,
	)
}

// GenerateMesh builds or rebuilds a quad/grid mesh for m from sprite s when
// m.IsDirty is true or when a non-custom mesh has no vertices yet.
func GenerateMesh(m *MeshComponent, s *SpriteComponent) {
	if !m.IsDirty && (len(m.Vertices) > 0 || m.MeshType == MeshTypeCustom) {
		return
	}

	rows := m.Rows
	if rows <= 0 {
		rows = 1
	}
	cols := m.Cols
	if cols <= 0 {
		cols = 1
	}

	var srcRect image.Rectangle
	if IsBoundEmpty(s.Bound) {
		srcRect = image.Rect(0, 0, s.Width, s.Height)
	} else {
		srcRect = image.Rect(int(s.Bound.Min.X), int(s.Bound.Min.Y), int(s.Bound.Max.X), int(s.Bound.Max.Y))
	}

	numVertices := (rows + 1) * (cols + 1)
	if cap(m.Vertices) < numVertices {
		m.Vertices = make([]ebiten.Vertex, numVertices)
	} else {
		m.Vertices = m.Vertices[:numVertices]
	}

	invCols := 1.0 / float32(cols)
	invRows := 1.0 / float32(rows)
	w := float32(s.Width)
	h := float32(s.Height)
	srcMinX := float32(srcRect.Min.X)
	srcMinY := float32(srcRect.Min.Y)
	srcDx := float32(srcRect.Dx())
	srcDy := float32(srcRect.Dy())

	for r := 0; r <= rows; r++ {
		fr := float32(r) * invRows
		vy := h * fr
		v := srcMinY + srcDy*fr
		rowOffset := r * (cols + 1)
		for c := 0; c <= cols; c++ {
			fc := float32(c) * invCols
			m.Vertices[rowOffset+c] = ebiten.Vertex{
				DstX:   w * fc,
				DstY:   vy,
				SrcX:   srcMinX + srcDx*fc,
				SrcY:   v,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			}
		}
	}

	numIndices := rows * cols * 6
	if cap(m.Indices) < numIndices {
		m.Indices = make([]uint16, numIndices)
	} else {
		m.Indices = m.Indices[:numIndices]
	}
	i := 0
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			topLeft := uint16(r*(cols+1) + c)
			topRight := topLeft + 1
			bottomLeft := uint16((r+1)*(cols+1) + c)
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

	if cap(m.BaseVertices) < numVertices {
		m.BaseVertices = make([]ebiten.Vertex, numVertices)
	} else {
		m.BaseVertices = m.BaseVertices[:numVertices]
	}
	copy(m.BaseVertices, m.Vertices)

	m.Rows = rows
	m.Cols = cols
	m.IsDirty = false
	m.cachedSrcRect = srcRect
}

// SyncMeshUVs ensures m has generated geometry and updates only SrcX/SrcY on
// m.Vertices and m.BaseVertices when the resolved source rectangle changes,
// preserving deformed DstX/DstY and vertex colors.
func SyncMeshUVs(m *MeshComponent, s *SpriteComponent, img *ebiten.Image) {
	if m == nil || s == nil {
		return
	}
	if m.IsDirty || (len(m.Vertices) == 0 && m.MeshType != MeshTypeCustom) {
		GenerateMesh(m, s)
	}
	if m.MeshType == MeshTypeCustom {
		return
	}
	targetRect := resolveSpriteSourceRect(s, img)
	rows := max(1, m.Rows)
	cols := max(1, m.Cols)
	numVertices := (rows + 1) * (cols + 1)
	if m.cachedSrcRect == targetRect || len(m.Vertices) != numVertices {
		return
	}

	invCols := 1.0 / float32(cols)
	invRows := 1.0 / float32(rows)
	srcMinX := float32(targetRect.Min.X)
	srcMinY := float32(targetRect.Min.Y)
	srcDx := float32(targetRect.Dx())
	srcDy := float32(targetRect.Dy())
	hasBase := len(m.BaseVertices) == numVertices

	for r := 0; r <= rows; r++ {
		v := srcMinY + srcDy*(float32(r)*invRows)
		rowOffset := r * (cols + 1)
		for c := 0; c <= cols; c++ {
			u := srcMinX + srcDx*(float32(c)*invCols)
			idx := rowOffset + c
			m.Vertices[idx].SrcX = u
			m.Vertices[idx].SrcY = v
			if hasBase {
				m.BaseVertices[idx].SrcX = u
				m.BaseVertices[idx].SrcY = v
			}
		}
	}
	m.cachedSrcRect = targetRect
}

// Bind adds a new input binding for a specific action.
// primary and modifiers may be ebiten.Key, ebiten.MouseButton,
// ebiten.GamepadButton, ebiten.StandardGamepadButton, ebiten.StandardGamepadAxis, or InputCode.
func Bind(input *InputComponent, action Action, primary any, modifiers ...any) {
	if input.Bindings == nil {
		input.Bindings = make(map[Action][]KeyConfig)
	}
	input.Bindings[action] = append(input.Bindings[action], NewKeyConfig(primary, modifiers...))
}

// BatchBind replaces all existing bindings on input with a new binding table.
func BatchBind(input *InputComponent, bindings map[Action][]KeyConfig) {
	input.Bindings = make(map[Action][]KeyConfig, len(bindings))
	for a, configs := range bindings {
		copied := make([]KeyConfig, len(configs))
		copy(copied, configs)
		input.Bindings[a] = copied
	}
}

// NewKeyConfig builds a KeyConfig from Ebitengine input types or InputCode values.
func NewKeyConfig(key any, modifiers ...any) KeyConfig {
	res := KeyConfig{
		Primary: toInputCode(key),
	}
	if len(modifiers) > 0 {
		res.Modifiers = make([]InputCode, len(modifiers))
		for i, m := range modifiers {
			res.Modifiers[i] = toInputCode(m)
		}
	}
	return res
}

// toInputCode converts an Ebitengine input constant or InputCode into an InputCode.
func toInputCode(v any) InputCode {
	switch code := v.(type) {
	case InputCode:
		return code
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
		panic("katsu2d: unsupported input type")
	}
}

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
