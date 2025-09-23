package katsu2d

import (
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TextRenderSystem renders text components.
type TextRenderSystem struct{}

// NewTextRenderSystem creates a new TextRenderSystem.
func NewTextRenderSystem() *TextRenderSystem {
	return &TextRenderSystem{}
}

// Draw renders all text components in the world using their transforms.
func (self *TextRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTText, CTTransform)
	for _, entity := range entities {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		txt, _ := world.GetComponent(entity, CTText)
		textComp := txt.(*TextComponent)
		renderer.Flush()
		textComp.UpdateCache()
		op := &text.DrawOptions{}
		op.LineSpacing = textComp.LineSpacing

		switch textComp.Alignment {
		case TextAlignmentTopRight, TextAlignmentMiddleRight, TextAlignmentBottomRight:
			op.PrimaryAlign = text.AlignStart
		case TextAlignmentTopCenter, TextAlignmentMiddleCenter, TextAlignmentBottomCenter:
			op.PrimaryAlign = text.AlignCenter
		default:
			op.PrimaryAlign = text.AlignEnd
		}

		offsetX, offsetY := textComp.GetOffset()
		t.SetOffset(ebimath.V(offsetX, offsetY))
		op.GeoM = t.Transform.Matrix()
		op.ColorScale = utils.RGBAToColorScale(textComp.Color)
		text.Draw(renderer.screen, textComp.Caption, textComp.FontFace, op)
	}
}

// AnimationSystem updates animations.
type AnimationSystem struct{}

// NewAnimationSystem creates a new AnimationSystem.
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

// Update advances all active animations in the world by the given delta time.
func (self *AnimationSystem) Update(world *World, dt float64) {
	entities := world.Query(CTAnimation, CTSprite)
	for _, e := range entities {
		animAny, _ := world.GetComponent(e, CTAnimation)
		anim := animAny.(*AnimationComponent)
		sprAny, _ := world.GetComponent(e, CTSprite)
		spr := sprAny.(*SpriteComponent)

		if !anim.Active || len(anim.Frames) == 0 {
			continue
		}
		anim.Elapsed += dt
		if anim.Elapsed >= anim.Speed {
			anim.Elapsed -= anim.Speed
			nf := len(anim.Frames)
			switch anim.Mode {
			case AnimOnce:
				if anim.Current+1 >= nf {
					anim.Current = nf - 1
					anim.Active = false
				} else {
					anim.Current++
				}
			case AnimLoop:
				anim.Current++
				anim.Current %= nf
			case AnimBoomerang:
				if nf > 1 {
					if anim.Direction {
						anim.Current++
						if anim.Current >= nf-1 {
							anim.Current = nf - 1
							anim.Direction = false
						}
					} else {
						anim.Current--
						if anim.Current < 0 {
							anim.Current = 0
							anim.Direction = true
						}
					}
				} else {
					anim.Current = 0
					anim.Active = false
				}
			}
			frame := anim.Frames[anim.Current]
			spr.SrcRect = &frame
		}
	}
}

// SpriteRenderSystem renders sprite components.
type SpriteRenderSystem struct {
	world             *World
	tm                *TextureManager
	drawableEntities  []Entity
	lastFrameEntities map[Entity]struct{}

	zSortNeeded bool
}

// NewSpriteRenderSystem creates a new SpriteRenderSystem with the given texture manager.
func NewSpriteRenderSystem(world *World, tm *TextureManager) *SpriteRenderSystem {
	srs := &SpriteRenderSystem{
		world:             world,
		tm:                tm,
		lastFrameEntities: make(map[Entity]struct{}),
	}
	eb := world.GetEventBus()
	if eb != nil {
		eb.Subscribe(TransformZDirtyEvent{}, srs.onTransformZDirty)
	}
	return srs
}

// onTransformZDirty sets zSortNeeded into true
func (self *SpriteRenderSystem) onTransformZDirty(_ interface{}) {
	self.zSortNeeded = true
}

// Update checks if a re-sort is needed for the drawables list.
func (self *SpriteRenderSystem) Update(world *World, dt float64) {
	includeComponents := []ComponentID{CTTransform, CTSprite}
	currentEntities := world.Query(includeComponents...)

	zSortNeeded := self.zSortNeeded || len(currentEntities) != len(self.lastFrameEntities)
	if !zSortNeeded && len(currentEntities) > 0 {
		for _, entity := range currentEntities {
			if _, ok := self.lastFrameEntities[entity]; !ok {
				zSortNeeded = true
				break
			}
		}
	}

	if zSortNeeded {
		self.drawableEntities = currentEntities
		sort.SliceStable(self.drawableEntities, func(i, j int) bool {
			t1Any, _ := world.GetComponent(self.drawableEntities[i], CTTransform)
			t1 := t1Any.(*TransformComponent)
			t2Any, _ := world.GetComponent(self.drawableEntities[j], CTTransform)
			t2 := t2Any.(*TransformComponent)
			return t1.Z < t2.Z
		})
		self.zSortNeeded = false
	}

	self.lastFrameEntities = make(map[Entity]struct{}, len(currentEntities))
	for _, entity := range currentEntities {
		self.lastFrameEntities[entity] = struct{}{}
	}
}

// Draw renders all sprites in the world, using the pre-sorted list.
func (self *SpriteRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	for _, entity := range self.drawableEntities {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		if s.Dirty {
			s.GenerateMesh()
		}

		worldVertices := make([]ebiten.Vertex, len(s.Vertices))
		transformMatrix := t.Matrix()
		for i, v := range s.Vertices {
			v.ColorR = float32(s.Color.R) / 255
			v.ColorG = float32(s.Color.G) / 255
			v.ColorB = float32(s.Color.B) / 255
			v.ColorA = float32(s.Color.A) / 255 * s.Opacity
			vx, vy := (&transformMatrix).Apply(float64(v.DstX), float64(v.DstY))
			v.DstX = float32(vx)
			v.DstY = float32(vy)
			worldVertices[i] = v
		}
		renderer.AddCustomMeshes(worldVertices, s.Indices, img)
	}
}
