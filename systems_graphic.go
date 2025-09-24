package katsu2d

import (
	"sort"
	"sync"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/edwinsyarief/lazyecs"
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
func (self *TextRenderSystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	query := world.Query(CTText, CTTransform)
	for query.Next() {
		renderer.Flush()
		txts, _ := lazyecs.GetComponentSlice[TextComponent](query)
		tranforms, _ := lazyecs.GetComponentSlice[TransformComponent](query)

		for i, txt := range txts {
			t := tranforms[i]
			txt.UpdateCache()
			op := &text.DrawOptions{}
			op.LineSpacing = txt.LineSpacing

			switch txt.Alignment {
			case TextAlignmentTopRight, TextAlignmentMiddleRight, TextAlignmentBottomRight:
				op.PrimaryAlign = text.AlignStart
			case TextAlignmentTopCenter, TextAlignmentMiddleCenter, TextAlignmentBottomCenter:
				op.PrimaryAlign = text.AlignCenter
			default:
				op.PrimaryAlign = text.AlignEnd
			}

			offsetX, offsetY := txt.GetOffset()
			t.SetOffset(ebimath.V(offsetX, offsetY))
			op.GeoM = t.Transform.Matrix()
			op.ColorScale = utils.RGBAToColorScale(txt.Color)
			text.Draw(renderer.screen, txt.Caption, txt.FontFace, op)
		}
	}
}

// AnimationSystem updates animations.
type AnimationSystem struct{}

// NewAnimationSystem creates a new AnimationSystem.
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

// Update advances all active animations in the world by the given delta time.
func (self *AnimationSystem) Update(world *lazyecs.World, dt float64) {
	query := world.Query(CTAnimation, CTSprite)
	for query.Next() {
		entities := query.Entities()
		for _, e := range entities {
			anim, _ := lazyecs.GetComponent[AnimationComponent](world, e)
			spr, _ := lazyecs.GetComponent[SpriteComponent](world, e)

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
}

// SpriteRenderSystem renders sprite components.
type SpriteRenderSystem struct {
	tm                *TextureManager
	drawableEntities  []lazyecs.Entity
	transforms        map[lazyecs.Entity]TransformComponent
	sprites           map[lazyecs.Entity]SpriteComponent
	lastFrameEntities map[lazyecs.Entity]struct{}
	zSortNeeded       bool

	once sync.Once
}

// NewSpriteRenderSystem creates a new SpriteRenderSystem with the given texture manager.
func NewSpriteRenderSystem(tm *TextureManager) *SpriteRenderSystem {
	srs := &SpriteRenderSystem{
		tm:                tm,
		lastFrameEntities: make(map[lazyecs.Entity]struct{}),
	}

	return srs
}

// onTransformZDirty sets zSortNeeded into true
func (self *SpriteRenderSystem) onTransformZDirty(_ interface{}) {
	self.zSortNeeded = true
}

// Update checks if a re-sort is needed for the drawables list.
func (self *SpriteRenderSystem) Update(world *lazyecs.World, dt float64) {
	self.once.Do(func() {
		eb := GetEventBus(world)
		if eb != nil {
			eb.Subscribe(TransformZDirtyEvent{}, self.onTransformZDirty)
		}
	})

	self.transforms = make(map[lazyecs.Entity]TransformComponent)
	self.sprites = make(map[lazyecs.Entity]SpriteComponent)

	query := world.Query(CTTransform, CTSprite)
	currentEntities := make([]lazyecs.Entity, 0)
	for query.Next() {
		currentEntities = append(currentEntities, query.Entities()...)
		transforms, _ := lazyecs.GetComponentSlice[TransformComponent](query)
		sprites, _ := lazyecs.GetComponentSlice[SpriteComponent](query)

		for i, entity := range query.Entities() {
			self.transforms[entity] = transforms[i]
			self.sprites[entity] = sprites[i]
		}
	}

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
			t1 := self.transforms[self.drawableEntities[i]]
			t2 := self.transforms[self.drawableEntities[j]]
			return t1.Z < t2.Z
		})
		self.zSortNeeded = false
	}

	self.lastFrameEntities = make(map[lazyecs.Entity]struct{}, len(currentEntities))
	for _, entity := range currentEntities {
		self.lastFrameEntities[entity] = struct{}{}
	}
}

// Draw renders all sprites in the world, using the pre-sorted list.
func (self *SpriteRenderSystem) Draw(world *lazyecs.World, renderer *BatchRenderer) {
	for _, entity := range self.drawableEntities {
		t := self.transforms[entity]
		s := self.sprites[entity]

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
