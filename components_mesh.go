package katsu2d

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
}
