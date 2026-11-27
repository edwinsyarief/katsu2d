package katsu2d

import "image"

type MeshType int

const (
	// MeshTypeQuad represents a simple quad mesh.
	MeshTypeQuad MeshType = iota
	// MeshTypeGrid represents a grid mesh.
	MeshTypeGrid
	// MeshTypeCustom represents a custom mesh defined by the user.
	MeshTypeCustom
)

// MeshTextureMode defines how the texture is applied to the mesh.
type MeshTextureMode int

const (
	// SpriteTextureModeStretch stretches the texture to fit the entire mesh.
	SpriteTextureModeStretch MeshTextureMode = iota
	// SpriteTextureModeTile is not yet implemented.
	SpriteTextureModeTile
)

type MeshComponent struct {
	BaseVertices, Vertices Vertices
	Indices                Indices
	Rows, Cols             int
	MeshType               MeshType
	TexMode                MeshTextureMode
	IsDirty                bool
	cachedSrcRect          image.Rectangle
}

// NewGridMeshComponent creates a dirty grid MeshComponent with the given row and column counts.
func NewGridMeshComponent(rows, cols int) MeshComponent {
	return MeshComponent{
		Rows:     rows,
		Cols:     cols,
		MeshType: MeshTypeGrid,
		IsDirty:  true,
	}
}
