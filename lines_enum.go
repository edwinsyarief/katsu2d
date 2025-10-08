package katsu2d

// LineJointMode defines how line segments are joined together at vertices
type LineJointMode int

const (
	// LineJointSharp creates sharp corners at line joints
	LineJointSharp LineJointMode = iota
	// LineJointBevel creates flattened corners at line joints
	LineJointBevel
	// LineJointRound creates rounded corners at line joints
	LineJointRound
)

// LineCapMode defines how the endpoints of lines are rendered
type LineCapMode int

const (
	// LineCapNone leaves line endpoints without any cap
	LineCapNone LineCapMode = iota
	// LineCapBox adds a square cap at line endpoints
	LineCapBox
	// LineCapRound adds a rounded cap at line endpoints
	LineCapRound
)

// LineTextureMode defines how textures are applied along a line
type LineTextureMode int

const (
	// LineTextureNone renders the line without any texture
	LineTextureNone LineTextureMode = iota
	// LineTextureTile repeats the texture along the line length
	LineTextureTile
	// LineTextureStretch stretches a single texture instance across the entire line
	LineTextureStretch
)
