package katsu2d

import (
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/utils"
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
		textComp.updateCache()
		op := &text.DrawOptions{}
		op.LineSpacing = textComp.LineSpacing()

		switch textComp.Alignment {
		case TextAlignmentTopRight:
		case TextAlignmentMiddleRight:
		case TextAlignmentBottomRight:
			op.PrimaryAlign = text.AlignStart
		case TextAlignmentTopCenter:
		case TextAlignmentMiddleCenter:
		case TextAlignmentBottomCenter:
			op.PrimaryAlign = text.AlignCenter
		default:
			op.PrimaryAlign = text.AlignStart
		}

		op.GeoM = t.Transform.Matrix()
		offsetX, offsetY := textComp.GetOffset()
		op.GeoM.Translate(offsetX, offsetY)
		op.ColorScale = utils.RGBAToColorScale(textComp.Color)
		text.Draw(renderer.screen, textComp.Caption, textComp.fontFace, op)
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
				// --- FIX: Handle single-frame boomerang gracefully ---
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
			spr.SrcX = float32(frame.Min.X)
			spr.SrcY = float32(frame.Min.Y)
			spr.SrcW = float32(frame.Dx())
			spr.SrcH = float32(frame.Dy())
		}
	}
}

// SpriteRenderSystem renders sprite components.
type SpriteRenderSystem struct {
	tm *TextureManager
	// The `drawableEntities` slice holds the pre-sorted list of entities.
	drawableEntities []Entity
	// A map to quickly track the entities from the last frame.
	lastFrameEntities map[Entity]struct{}
}

// NewSpriteRenderSystem creates a new SpriteRenderSystem with the given texture manager.
func NewSpriteRenderSystem(tm *TextureManager) *SpriteRenderSystem {
	return &SpriteRenderSystem{
		tm:                tm,
		lastFrameEntities: make(map[Entity]struct{}),
	}
}

// Update checks if a re-sort is needed for the drawables list.
// This should be run before the Draw method.
func (self *SpriteRenderSystem) Update(world *World, dt float64) {
	// Query for the current set of entities.
	currentEntities := world.Query(CTSprite, CTTransform)

	// A sort is needed if the world has been explicitly marked as dirty,
	// or if the number of entities has changed.
	// We no longer need to loop through all entities to check for dirty flags.
	zSortNeeded := world.zSortNeeded || len(currentEntities) != len(self.lastFrameEntities)

	// If the lengths are the same, we still need to check if the set of
	// entities has changed (e.g., one was destroyed, another was created).
	if !zSortNeeded && len(currentEntities) == len(self.lastFrameEntities) {
		for _, entity := range currentEntities {
			if _, ok := self.lastFrameEntities[entity]; !ok {
				zSortNeeded = true
				break
			}
		}
	}

	if zSortNeeded {
		// Rebuild the drawableEntities slice from the current world state.
		self.drawableEntities = currentEntities

		// Sort the slice based on Z-index.
		sort.SliceStable(self.drawableEntities, func(i, j int) bool {
			t1Any, _ := world.GetComponent(self.drawableEntities[i], CTTransform)
			t1 := t1Any.(*TransformComponent)
			t2Any, _ := world.GetComponent(self.drawableEntities[j], CTTransform)
			t2 := t2Any.(*TransformComponent)
			return t1.Z < t2.Z
		})

		// Reset the world's sort needed flag since we just sorted.
		world.zSortNeeded = false
	}

	// Update the entity map for the next frame's check.
	// We do this unconditionally so that entity adds/removes are tracked correctly.
	self.lastFrameEntities = make(map[Entity]struct{}, len(currentEntities))
	for _, entity := range currentEntities {
		self.lastFrameEntities[entity] = struct{}{}
	}
}

// Draw renders all sprites in the world, using the pre-sorted list.
func (self *SpriteRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	// The drawableEntities list is already sorted by the Update method.
	for _, entity := range self.drawableEntities {
		tx, _ := world.GetComponent(entity, CTTransform)
		t := tx.(*TransformComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		img := self.tm.Get(s.TextureID)
		if img == nil {
			continue
		}

		imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
		srcX, srcY, srcW, srcH := s.GetSourceRect(float32(imgW), float32(imgH))
		destW32, destH32 := s.GetDestSize(srcW, srcH)
		destW, destH := float64(destW32), float64(destH32)

		effColor := s.Color
		effColor.A = uint8(float32(s.Color.A) * s.Opacity)

		realPos := ebimath.V2(0).Apply(t.Matrix())
		if !t.Origin().IsZero() {
			realPos = realPos.Sub(t.Origin())
		}

		renderer.DrawQuad(
			realPos,
			t.Scale(),
			t.Rotation(),
			img,
			effColor,
			srcX, srcY, srcX+srcW, srcY+srcH,
			destW, destH,
		)
	}
}
