package line

type LineJointMode int

const (
	LineJointSharp LineJointMode = iota
	LineJointBevel
	LineJointRound
)

type LineCapMode int

const (
	LineCapNone LineCapMode = iota
	LineCapBox
	LineCapRound
)

type LineTextureMode int

const (
	LineTextureNone LineTextureMode = iota
	LineTextureTile
	LineTextureStretch
)
