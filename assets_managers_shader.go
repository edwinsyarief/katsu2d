package katsu2d

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// ShaderManager manages loading, compiling, and retrieving Kage shaders by ID.
type ShaderManager struct {
	shaders []*ebiten.Shader
}

// NewShaderManager creates an empty ShaderManager.
func NewShaderManager() *ShaderManager {
	return &ShaderManager{
		shaders: make([]*ebiten.Shader, 0),
	}
}

// Load compiles a Kage shader from disk and returns its ID.
func (sm *ShaderManager) Load(path string) (int, error) {
	b, err := readFileErr(path)
	if err != nil {
		return -1, fmt.Errorf("failed to read shader %q: %w", path, err)
	}
	return sm.LoadBytes(b)
}

// LoadEmbedded compiles a Kage shader from the embedded filesystem and returns its ID.
func (sm *ShaderManager) LoadEmbedded(path string) int {
	id, err := sm.LoadBytes(openEmbeddedFile(path))
	if err != nil {
		return -1
	}
	return id
}

// LoadFromAssetPacker compiles a Kage shader from the asset bundle and returns its ID.
func (sm *ShaderManager) LoadFromAssetPacker(path string) int {
	id, err := sm.LoadBytes(openBundledFile(path))
	if err != nil {
		return -1
	}
	return id
}

// LoadBytes compiles a Kage shader from raw source bytes and returns its ID.
func (sm *ShaderManager) LoadBytes(content []byte) (int, error) {
	shader, err := ebiten.NewShader(content)
	if err != nil {
		return -1, fmt.Errorf("failed to compile shader: %w", err)
	}
	return sm.Add(shader), nil
}

// Get retrieves a shader by ID, or nil if out of range.
func (sm *ShaderManager) Get(id int) *ebiten.Shader {
	if id < 0 || id >= len(sm.shaders) {
		return nil
	}
	return sm.shaders[id]
}

// Add registers a pre-compiled *ebiten.Shader and returns its ID.
func (sm *ShaderManager) Add(shader *ebiten.Shader) int {
	id := len(sm.shaders)
	sm.shaders = append(sm.shaders, shader)
	return id
}
