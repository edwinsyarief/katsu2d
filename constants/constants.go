package constants

// Component type IDs
const (
	InvalidComponentID = iota
	ComponentTransform
	ComponentDrawable
	ComponentDrawableBatch
	ComponentUpdatable
	ComponentTag
	ComponentCooldown
	ComponentCustom
)

// ECS constants
const (
	MaxEntities     = 100000
	MaxVertices     = 65534                 // For batched rendering
	MaxIndices      = (MaxVertices / 4) * 6 // 6 indices per quad
	InvalidEntityID = 0
)

// Scene constants
const (
	DefaultSceneName = "default"
)
