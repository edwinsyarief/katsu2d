package katsu2d

import (
	"image"
	"image/color"
	"sync"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// ComponentPool implements object pooling for game components to reduce memory allocations
// and garbage collection overhead.

// --- SpriteComponent Pool ---
var spriteComponentPool = sync.Pool{
	New: func() interface{} {
		return NewSpriteComponent(0, image.Rect(0, 0, 1, 1))
	},
}

// GetSpriteComponent retrieves a SpriteComponent from the pool
func GetSpriteComponent() *SpriteComponent {
	s := spriteComponentPool.Get().(*SpriteComponent)
	s.TextureID = 0
	s.SrcRect = nil
	s.DstW = 0
	s.DstH = 0
	s.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	s.Opacity = 1.0
	return s
}

// PutSpriteComponent returns a SpriteComponent to the pool for reuse
func PutSpriteComponent(s *SpriteComponent) {
	spriteComponentPool.Put(s)
}

// --- TransformComponent Pool ---
var transformComponentPool = sync.Pool{
	New: func() interface{} {
		return NewTransformComponent()
	},
}

// GetTransformComponent retrieves a TransformComponent from the pool and resets its state
func GetTransformComponent() *TransformComponent {
	t := transformComponentPool.Get().(*TransformComponent)
	t.SetPosition(ebimath.Vector{})
	t.SetScale(ebimath.Vector{X: 1, Y: 1})
	t.SetRotation(0)
	t.Z = 0
	t.SetOrigin(ebimath.Vector{})
	return t
}

// PutTransformComponent returns a TransformComponent to the pool for reuse
func PutTransformComponent(t *TransformComponent) {
	transformComponentPool.Put(t)
}

// --- ParticleComponent Pool ---
var particleComponentPool = sync.Pool{
	New: func() interface{} {
		return &ParticleComponent{}
	},
}

// GetParticleComponent retrieves a ParticleComponent from the pool
func GetParticleComponent() *ParticleComponent {
	return particleComponentPool.Get().(*ParticleComponent)
}

// PutParticleComponent returns a ParticleComponent to the pool for reuse
func PutParticleComponent(p *ParticleComponent) {
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
