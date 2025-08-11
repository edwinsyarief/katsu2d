package katsu2d

import (
	"math"
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/hajimehoshi/ebiten/v2"
)

// System defines the interface for game systems that update or draw.
type System interface {
	Update(world *World, dt float64)
	Draw(world *World, screen *ebiten.Image)
}

// RenderSystem handles batched sprite (quad) rendering using DrawTriangles.
type RenderSystem struct {
	tm *TextureManager
}

// NewRenderSystem creates a new render system with the given texture manager.
func NewRenderSystem(tm *TextureManager) *RenderSystem {
	return &RenderSystem{tm: tm}
}

// Update does nothing for this system.
func (s *RenderSystem) Update(*World, float64) {}

// RenderItem is used for sorting entities by Z-index.
type RenderItem struct {
	Entity Entity
	Z      float64
}

// Draw renders all sprites, sorted by Z, batched by texture.
func (s *RenderSystem) Draw(w *World, screen *ebiten.Image) {
	entities := w.QueryAll(CTTransform, CTSprite)
	var items []RenderItem
	for _, e := range entities {
		pos := w.GetComponent(e, CTTransform).(*Transform)
		items = append(items, RenderItem{Entity: e, Z: pos.Z})
	}

	// Sort by Z for correct draw order
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Z < items[j].Z
	})

	// Batching
	var verts []ebiten.Vertex
	var indices []uint16
	currentTex := -1

	// drawBatch flushes the current vertex batch to the screen.
	drawBatch := func() {
		if len(verts) == 0 {
			return
		}
		img := s.tm.Get(currentTex)
		screen.DrawTriangles(verts, indices, img, nil)
		verts = verts[:0]
		indices = indices[:0]
	}

	for _, item := range items {
		e := item.Entity
		pos := w.GetComponent(e, CTTransform).(*Transform)
		spr := w.GetComponent(e, CTSprite).(*Sprite)

		texID := spr.TextureID
		if texID != currentTex {
			drawBatch()
			currentTex = texID
		}

		img := s.tm.Get(texID)
		iw, ih := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())

		// Get adjusted sprite dimensions
		sx, sy, sw, sh := spr.GetSourceRect(iw, ih)
		dw, dh := spr.GetDestSize(sw, sh)

		hw := dw / 2
		hh := dh / 2

		cos := float32(math.Cos(pos.Rotation()))
		sin := float32(math.Sin(pos.Rotation()))

		lx0, ly0 := -hw, -hh
		lx1, ly1 := hw, hh

		lx0 *= float32(pos.Scale().X)
		lx1 *= float32(pos.Scale().X)
		ly0 *= float32(pos.Scale().Y)
		ly1 *= float32(pos.Scale().Y)

		// Rotated coordinates
		rx0 := lx0*cos - ly0*sin
		ry0 := lx0*sin + ly0*cos
		rx1 := lx1*cos - ly0*sin
		ry1 := lx1*sin + ly0*cos
		rx2 := lx0*cos - ly1*sin
		ry2 := lx0*sin + ly1*cos
		rx3 := lx1*cos - ly1*sin
		ry3 := lx1*sin + ly1*cos

		vec := ebimath.V2(0).Apply(pos.Matrix())
		px, py := float32(vec.X), float32(vec.Y)

		sx0, sy0 := sx, sy
		sx1, sy1 := sx+sw, sy+sh

		// Color (default to white if fully transparent/zero)
		r, g, b, a := float32(1), float32(1), float32(1), float32(1)
		if spr.Color.A != 0 {
			rr, gg, bb, aa := spr.Color.RGBA()
			r = float32(rr) / 0xFFFF
			g = float32(gg) / 0xFFFF
			b = float32(bb) / 0xFFFF
			a = float32(aa) / 0xFFFF
		}

		baseIdx := len(verts)
		verts = append(verts,
			ebiten.Vertex{DstX: px + rx0, DstY: py + ry0, SrcX: sx0, SrcY: sy0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			ebiten.Vertex{DstX: px + rx1, DstY: py + ry1, SrcX: sx1, SrcY: sy0, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			ebiten.Vertex{DstX: px + rx2, DstY: py + ry2, SrcX: sx0, SrcY: sy1, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
			ebiten.Vertex{DstX: px + rx3, DstY: py + ry3, SrcX: sx1, SrcY: sy1, ColorR: r, ColorG: g, ColorB: b, ColorA: a},
		)
		bi := uint16(baseIdx)
		indices = append(indices, bi, bi+1, bi+2, bi+1, bi+3, bi+2)
	}

	drawBatch()
}

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

func (s *TweenSystem) Update(w *World, dt float64) {
	// Standalone tweens
	entities := w.QueryAll(CTTween)
	for _, e := range entities {
		tw := w.GetComponent(e, CTTween).(*Tween)
		tw.Update(float32(dt))
	}

	// Sequences
	entities = w.QueryAll(CTSequence)
	for _, e := range entities {
		seq := w.GetComponent(e, CTSequence).(*Sequence)
		seq.Update(float32(dt))
	}
}

func (s *TweenSystem) Draw(*World, *ebiten.Image) {}

// AnimationSystem updates animations.
type AnimationSystem struct{}

func (s *AnimationSystem) Update(w *World, dt float64) {
	entities := w.QueryAll(CTAnimation, CTSprite)
	for _, e := range entities {
		anim := w.GetComponent(e, CTAnimation).(*Animation)
		spr := w.GetComponent(e, CTSprite).(*Sprite)
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

func (s *AnimationSystem) Draw(*World, *ebiten.Image) {}

type FadeOverlaySystem struct{}

func (self *FadeOverlaySystem) Update(w *World, dt float64) {
	entities := w.QueryAll(CTFadeOverlay)
	for _, e := range entities {
		fade := w.GetComponent(e, CTFadeOverlay).(*overlays.FadeOverlay)
		fade.Update(dt)
	}
}

func (self *FadeOverlaySystem) Draw(w *World, screen *ebiten.Image) {
	entities := w.QueryAll(CTFadeOverlay)
	for _, e := range entities {
		fade := w.GetComponent(e, CTFadeOverlay).(*overlays.FadeOverlay)
		fade.Draw(screen)
	}
}

type CinematicOverlaySystem struct{}

func (self *CinematicOverlaySystem) Update(w *World, dt float64) {
	entities := w.QueryAll(CTCinematicOverlay)
	for _, e := range entities {
		cinematic := w.GetComponent(e, CTCinematicOverlay).(*overlays.CinematicOverlay)
		cinematic.Update(dt)
	}
}

func (self *CinematicOverlaySystem) Draw(w *World, screen *ebiten.Image) {
	entities := w.QueryAll(CTCinematicOverlay)
	for _, e := range entities {
		cinematic := w.GetComponent(e, CTCinematicOverlay).(*overlays.CinematicOverlay)
		cinematic.Draw(screen)
	}
}

type CooldownSystem struct{}

func (self *CooldownSystem) Update(w *World, dt float64) {
	entities := w.QueryAll(CTCooldown)
	for _, e := range entities {
		cm := w.GetComponent(e, CTCooldown).(*managers.CooldownManager)
		cm.Update(dt)
	}
}

func (self *CooldownSystem) Draw(*World, *ebiten.Image) {}

type DelayManager struct{}

func (self *DelayManager) Update(w *World, dt float64) {
	entities := w.QueryAll(CTDelayer)
	for _, e := range entities {
		delay := w.GetComponent(e, CTDelayer).(*managers.DelayManager)
		delay.Update(dt)
	}
}

func (self *DelayManager) Draw(*World, *ebiten.Image) {}
