package katsu2d

import (
	"image/color"
	"sync"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// ComponentPool implements object pooling for game components to reduce memory allocations
// and garbage collection overhead. Each component type has its own sync.Pool implementation
// with appropriate creation and reset logic.

// --- SpriteComponent Pool ---
// spriteComponentPool maintains a pool of reusable SpriteComponents
var spriteComponentPool = sync.Pool{
	New: func() interface{} {
		return NewSpriteComponent(0, 0, 0)
	},
}

// GetSpriteComponent retrieves a SpriteComponent from the pool
// Returns: A recycled or new SpriteComponent instance
func GetSpriteComponent() *SpriteComponent {
	s := spriteComponentPool.Get().(*SpriteComponent)
	// Reset all component properties to default values
	s.TextureID = 0
	s.SrcX = 0
	s.SrcY = 0
	s.SrcW = 0
	s.SrcH = 0
	s.DstW = 0
	s.DstH = 0
	s.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	s.Opacity = 1.0
	return s
}

// PutSpriteComponent returns a SpriteComponent to the pool for reuse
// Parameters:
//   - s: The SpriteComponent to be recycled
func PutSpriteComponent(s *SpriteComponent) {
	spriteComponentPool.Put(s)
}

// --- TransformComponent Pool ---
// transformComponentPool maintains a pool of reusable TransformComponents
var transformComponentPool = sync.Pool{
	New: func() interface{} {
		return NewTransformComponent()
	},
}

// GetTransformComponent retrieves a TransformComponent from the pool and resets its state
// Returns: A recycled or new TransformComponent instance with default values
func GetTransformComponent() *TransformComponent {
	t := transformComponentPool.Get().(*TransformComponent)
	// Reset transform to default state
	t.SetPosition(ebimath.Vector{})
	t.SetScale(ebimath.Vector{X: 1, Y: 1})
	t.SetRotation(0)
	t.Z = 0
	t.SetOrigin(ebimath.Vector{})
	return t
}

// PutTransformComponent returns a TransformComponent to the pool for reuse
// Parameters:
//   - t: The TransformComponent to be recycled
func PutTransformComponent(t *TransformComponent) {
	transformComponentPool.Put(t)
}

// --- ParticleComponent Pool ---
// particleComponentPool maintains a pool of reusable ParticleComponents
var particleComponentPool = sync.Pool{
	New: func() interface{} {
		return &ParticleComponent{}
	},
}

// GetParticleComponent retrieves a ParticleComponent from the pool
// Returns: A recycled or new ParticleComponent instance
func GetParticleComponent() *ParticleComponent {
	return particleComponentPool.Get().(*ParticleComponent)
}

// PutParticleComponent returns a ParticleComponent to the pool for reuse
// Parameters:
//   - p: The ParticleComponent to be recycled
func PutParticleComponent(p *ParticleComponent) {
	// Reset all particle properties to default values
	p.Velocity = ebimath.ZeroVector
	p.Lifetime = 0
	p.TotalLifetime = 0
	p.InitialColor = color.RGBA{}
	p.TargetColor = color.RGBA{}
	p.InitialScale = 0
	p.TargetScale = 0
	p.InitialRotation = 0
	p.TargetRotation = 0
	p.Gravity = ebimath.ZeroVector
	particleComponentPool.Put(p)
}
