package katsu2d

import (
	"image/color"
	"math"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
)

// --- SYSTEMS ---

// UpdateSystem is an interface for update logic.
type UpdateSystem interface {
	Update(*Engine, float64)
}

// DrawSystem is an interface for draw logic.
type DrawSystem interface {
	Draw(*Engine, *BatchRenderer)
}

// --- RENDERER ---

// BatchRenderer batches draw calls for performance.
type BatchRenderer struct {
	screen       *ebiten.Image
	vertices     []ebiten.Vertex
	indices      []uint16
	currentImage *ebiten.Image
}

// NewBatchRenderer creates a new batch renderer.
func NewBatchRenderer() *BatchRenderer {
	return &BatchRenderer{
		vertices: make([]ebiten.Vertex, 0, 4096),
		indices:  make([]uint16, 0, 6144),
	}
}

func (self *BatchRenderer) GetScreen() *ebiten.Image {
	return self.screen
}

// Begin prepares the renderer for a new frame.
func (self *BatchRenderer) Begin(screen *ebiten.Image) {
	self.screen = screen
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// Flush draws the current batch.
func (self *BatchRenderer) Flush() {
	if len(self.vertices) == 0 {
		return
	}
	self.screen.DrawTriangles(self.vertices, self.indices, self.currentImage, nil)
	self.vertices = self.vertices[:0]
	self.indices = self.indices[:0]
	self.currentImage = nil
}

// AddVertices adds custom vertices and indices to the batch.
func (self *BatchRenderer) AddVertices(verts []ebiten.Vertex, inds []uint16, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)
	self.vertices = append(self.vertices, verts...)
	for _, i := range inds {
		self.indices = append(self.indices, uint16(offset)+i)
	}
}

// DrawQuad draws a quad (sprite).
func (self *BatchRenderer) DrawQuad(pos, scale, offset, origin ebimath.Vector, rotation float64, img *ebiten.Image, clr color.RGBA) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img

	w, h := float64(img.Bounds().Dx())*scale.X, float64(img.Bounds().Dy())*scale.Y
	ox, oy := float64(origin.X), float64(origin.Y)

	p0 := ebimath.V(-ox, -oy)
	p1 := ebimath.V(w-ox, -oy)
	p2 := ebimath.V(w-ox, h-oy)
	p3 := ebimath.V(-ox, h-oy)

	if rotation != 0 {
		c, s := math.Cos(rotation), math.Sin(rotation)
		p0 = ebimath.V(p0.X*c-p0.Y*s, p0.X*s+p0.Y*c)
		p1 = ebimath.V(p1.X*c-p1.Y*s, p1.X*s+p1.Y*c)
		p2 = ebimath.V(p2.X*c-p2.Y*s, p2.X*s+p2.Y*c)
		p3 = ebimath.V(p3.X*c-p3.Y*s, p3.X*s+p3.Y*c)
	}

	// Correctly apply the offset parameter here, before adding the final position.
	p0 = p0.Add(offset)
	p1 = p1.Add(offset)
	p2 = p2.Add(offset)
	p3 = p3.Add(offset)

	p0 = p0.Add(pos)
	p1 = p1.Add(pos)
	p2 = p2.Add(pos)
	p3 = p3.Add(pos)

	minX, minY := float32(img.Bounds().Min.X), float32(img.Bounds().Min.Y)
	maxX, maxY := float32(img.Bounds().Max.X), float32(img.Bounds().Max.Y)

	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255

	vertIndex := len(self.vertices)
	self.vertices = append(self.vertices,
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p0.X)), DstY: utils.AdjustDestinationPixel(float32(p0.Y)), SrcX: minX, SrcY: minY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p1.X)), DstY: utils.AdjustDestinationPixel(float32(p1.Y)), SrcX: maxX, SrcY: minY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p2.X)), DstY: utils.AdjustDestinationPixel(float32(p2.Y)), SrcX: maxX, SrcY: maxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
		ebiten.Vertex{DstX: utils.AdjustDestinationPixel(float32(p3.X)), DstY: utils.AdjustDestinationPixel(float32(p3.Y)), SrcX: minX, SrcY: maxY, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
	)
	self.indices = append(self.indices, uint16(vertIndex), uint16(vertIndex+1), uint16(vertIndex+2), uint16(vertIndex), uint16(vertIndex+2), uint16(vertIndex+3))
}

// DrawTriangleStrip draws a triangle strip.
func (self *BatchRenderer) DrawTriangleStrip(verts []ebiten.Vertex, img *ebiten.Image) {
	if img != self.currentImage && self.currentImage != nil {
		self.Flush()
	}
	self.currentImage = img
	offset := len(self.vertices)
	self.vertices = append(self.vertices, verts...)
	for i := 0; i < len(verts)-2; i++ {
		a := uint16(offset + i)
		bb := uint16(offset + i + 1)
		c := uint16(offset + i + 2)
		if i%2 == 0 {
			self.indices = append(self.indices, a, bb, c)
		} else {
			self.indices = append(self.indices, a, c, bb)
		}
	}
}

// DrawText draws text, flushing the batch first to maintain render order.
// This is necessary because text.Draw is a separate operation from DrawTriangles.
func (self *BatchRenderer) DrawText(txt *Text, transform *ebimath.Transform) {
	self.Flush()
	txt.Draw(transform, self.screen)
}

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

func (self *TweenSystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)

	// - Scenes -
	self.update(engine.sm.current.World, dt)
}

func (self *TweenSystem) update(world *World, dt float64) {
	// Standalone tweens
	entities := world.Query(CTTween)
	for _, e := range entities {
		twAny, _ := world.GetComponent(e, CTTween)
		tw := twAny.(*tween.Tween)
		tw.Update(float32(dt))
	}

	// Standalone Sequences
	entities = world.Query(CTSequence)
	for _, e := range entities {
		seqAny, _ := world.GetComponent(e, CTSequence)
		seq := seqAny.(*tween.Sequence)
		seq.Update(float32(dt))
	}
}

// AnimationSystem updates animations.
type AnimationSystem struct{}

func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

func (self *AnimationSystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)
	// - Scene -
	self.update(engine.sm.current.World, dt)
}

func (self *AnimationSystem) update(world *World, dt float64) {
	entities := world.Query(CTAnimation, CTSprite)
	for _, e := range entities {
		animAny, _ := world.GetComponent(e, CTAnimation)
		anim := animAny.(*Animation)
		sprAny, _ := world.GetComponent(e, CTSprite)
		spr := sprAny.(*Sprite)

		if !anim.Active || len(anim.Frames) == 0 {
			continue
		}
		anim.Elapsed += dt
		if anim.Elapsed >= anim.Speed {
			anim.Elapsed -= anim.Speed
			if anim.Direction {
				anim.Current++
			} else {
				anim.Current--
			}
			nf := len(anim.Frames)
			switch anim.Mode {
			case AnimOnce:
				if anim.Current >= nf {
					anim.Current = nf - 1
					anim.Active = false
				}
			case AnimLoop:
				anim.Current %= nf
			case AnimBoomerang:
				if anim.Current >= nf {
					anim.Current = nf - 2
					anim.Direction = false
				} else if anim.Current < 0 {
					anim.Current = 1
					anim.Direction = true
				}
			}
			frame := anim.Frames[anim.Current]
			spr.SrcX = float32(frame.Min.X)
			spr.SrcY = float32(frame.Min.Y)
			spr.SrcW = float32(frame.Dx())
			spr.SrcH = float32(frame.Dy())
		}
	}
}

type FadeOverlaySystem struct{}

func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{}
}

func (self *FadeOverlaySystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)

	// - Scene -
	self.update(engine.sm.current.World, dt)
}

func (self *FadeOverlaySystem) update(world *World, dt float64) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Update(dt)
	}
}

func (self *FadeOverlaySystem) Draw(engine *Engine, renderer *BatchRenderer) {
	// - Scene -
	self.draw(engine.sm.current.World, renderer.screen)

	// - Engine -
	self.draw(engine.world, renderer.screen)
}

func (self *FadeOverlaySystem) draw(world *World, screen *ebiten.Image) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Draw(screen)
	}
}

type CinematicOverlaySystem struct{}

func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	return &CinematicOverlaySystem{}
}

func (self *CinematicOverlaySystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)

	// - Scene
	self.update(engine.sm.current.World, dt)
}

func (self *CinematicOverlaySystem) update(world *World, dt float64) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Update(dt)
	}
}

func (self *CinematicOverlaySystem) Draw(engine *Engine, renderer *BatchRenderer) {
	// - Scene -
	self.draw(engine.sm.current.World, renderer.screen)

	// - Engine -
	self.draw(engine.world, renderer.screen)
}

func (self *CinematicOverlaySystem) draw(world *World, screen *ebiten.Image) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Draw(screen)
	}
}

type CooldownSystem struct{}

func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

func (self *CooldownSystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)

	// - Scene -
	self.update(engine.sm.current.World, dt)
}

func (self *CooldownSystem) update(world *World, dt float64) {
	entities := world.Query(CTCooldown)
	for _, e := range entities {
		cmAny, _ := world.GetComponent(e, CTCooldown)
		cm := cmAny.(*managers.CooldownManager)
		cm.Update(dt)
	}
}

type DelaySystem struct{}

func NewDelaySystem() *DelaySystem {
	return &DelaySystem{}
}

func (self *DelaySystem) Update(engine *Engine, dt float64) {
	// - Engine -
	self.update(engine.world, dt)

	// - Scene -
	self.update(engine.sm.current.World, dt)
}

func (self *DelaySystem) update(w *World, dt float64) {
	entities := w.Query(CTDelayer)
	for _, e := range entities {
		delayAny, _ := w.GetComponent(e, CTDelayer)
		delay := delayAny.(*managers.DelayManager)
		delay.Update(dt)
	}
}

type TextRenderSystem struct{}

func NewTextRenderSystem() *TextRenderSystem {
	return &TextRenderSystem{}
}

func (self *TextRenderSystem) Draw(engine *Engine, renderer *BatchRenderer) {
	// - Scene -
	self.draw(engine.sm.current.World, renderer)

	// - Engine -
	self.draw(engine.world, renderer)
}

func (self *TextRenderSystem) draw(w *World, renderer *BatchRenderer) {
	for _, entity := range w.Query(CTText, CTTransform) {
		if tx, ok := w.GetComponent(entity, CTTransform); ok {
			t := tx.(*Transform)
			if txt, ok := w.GetComponent(entity, CTText); ok {
				text := txt.(*Text)
				text.Draw(t.Transform, renderer.GetScreen())
			}
		}
	}
}

type SpriteRenderSystem struct{}

func NewSpriteRenderSystem() *SpriteRenderSystem {
	return &SpriteRenderSystem{}
}

func (self *SpriteRenderSystem) Draw(engine *Engine, renderer *BatchRenderer) {
	// - Scene -
	self.draw(engine.sm.current.World, engine.tm, renderer)

	// - Engine -
	self.draw(engine.world, engine.tm, renderer)
}

func (self *SpriteRenderSystem) draw(w *World, tm *TextureManager, renderer *BatchRenderer) {
	for _, entity := range w.Query(CTSprite, CTTransform) {
		tx, _ := w.GetComponent(entity, CTTransform)
		t := tx.(*Transform)
		sprite, _ := w.GetComponent(entity, CTSprite)
		s := sprite.(*Sprite)

		img := tm.Get(s.TextureID)
		renderer.DrawQuad(
			t.Position(), t.Scale(), t.Offset(), t.Origin(), t.Rotation(),
			img, s.Color)
	}
}
