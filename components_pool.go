package katsu2d

import (
	"image/color"
	"sync"

	ebimath "github.com/edwinsyarief/ebi-math"
)

// This file contains the component pooling logic for efficiency.
// It's a good practice to keep this separate from the core systems.

// --- SpriteComponent Pool ---
var spriteComponentPool = sync.Pool{
	New: func() interface{} {
		return NewSpriteComponent(0, 0, 0)
	},
}

// GetSpriteComponent retrieves a SpriteComponent from the pool.
func GetSpriteComponent() *SpriteComponent {
	return spriteComponentPool.Get().(*SpriteComponent)
}

// PutSpriteComponent returns a SpriteComponent to the pool.
func PutSpriteComponent(s *SpriteComponent) {
	// Reset the component to its default state before returning it to the pool.
	s.TextureID = 0
	s.SrcX = 0
	s.SrcY = 0
	s.SrcW = 0
	s.SrcH = 0
	s.DstW = 0
	s.DstH = 0
	s.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	s.Opacity = 1.0
	spriteComponentPool.Put(s)
}

// --- TransformComponent Pool ---
// (Dummy implementation, assuming it exists elsewhere)
var transformComponentPool = sync.Pool{
	New: func() interface{} {
		return NewTransformComponent()
	},
}

func GetTransformComponent() *TransformComponent {
	t := transformComponentPool.Get().(*TransformComponent)
	// Reset to default
	t.SetPosition(ebimath.Vector{})
	t.SetScale(ebimath.Vector{X: 1, Y: 1})
	t.SetRotation(0)
	t.Z = 0
	t.SetOrigin(ebimath.Vector{})
	return t
}

func PutTransformComponent(t *TransformComponent) {
	transformComponentPool.Put(t)
}

// --- ParticleComponent Pool ---
// (Dummy implementation, assuming it exists elsewhere)
var particleComponentPool = sync.Pool{
	New: func() interface{} {
		return &ParticleComponent{}
	},
}

func GetParticleComponent() *ParticleComponent {
	return particleComponentPool.Get().(*ParticleComponent)
}

func PutParticleComponent(p *ParticleComponent) {
	// Reset component to default state
	p.Velocity = ebimath.Vector{}
	p.Lifetime = 0
	p.TotalLifetime = 0
	p.InitialColor = color.RGBA{}
	p.TargetColor = color.RGBA{}
	p.InitialScale = 0
	p.TargetScale = 0
	p.InitialRotation = 0
	p.TargetRotation = 0
	particleComponentPool.Put(p)
}
