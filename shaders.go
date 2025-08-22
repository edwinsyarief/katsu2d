package katsu2d

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
)

// ShaderManager manages loading and retrieving shaders.
type ShaderManager struct {
	shaders []*ebiten.Shader
}

// NewShaderManager creates a manager with an empty shader list.
func NewShaderManager() *ShaderManager {
	sm := &ShaderManager{
		shaders: make([]*ebiten.Shader, 0),
	}
	return sm
}

// Load loads a shader from file and returns its ID.
func (self *ShaderManager) Load(path string) (int, error) {
	b := readFile(path)
	id := self.fromByte(b)
	if id == -1 {
		return -1, errors.New("failed to load shader from file: " + path)
	}

	return id, nil
}

// LoadEmbedded loads an shader from embedded file and returns its ID.
func (self *ShaderManager) LoadEmbedded(path string) int {
	b := openEmbeddedFile(path)
	return self.fromByte(b)
}

func (self *ShaderManager) LoadFromAssetPacker(path string) int {
	b := openBundledFile(path)
	return self.fromByte(b)
}

func (self *ShaderManager) fromByte(content []byte) int {
	shader, err := ebiten.NewShader(content)
	if err != nil {
		return -1
	}
	id := len(self.shaders)
	self.shaders = append(self.shaders, shader)
	return id
}

// Get retrieves a shader by ID
func (self *ShaderManager) Get(id int) *ebiten.Shader {
	if id < 0 || id >= len(self.shaders) {
		return nil
	}
	return self.shaders[id]
}

// Add adds a pre-existing *ebiten.Shader to the manager and returns its new ID.
func (self *ShaderManager) Add(shader *ebiten.Shader) int {
	id := len(self.shaders)
	self.shaders = append(self.shaders, shader)
	return id
}
