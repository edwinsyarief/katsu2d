package katsu2d

import (
	"math"
	"sort"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d/managers"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/edwinsyarief/katsu2d/tween"
	"github.com/edwinsyarief/katsu2d/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TweenSystem updates tweens and sequences.
type TweenSystem struct{}

// NewTweenSystem creates a new TweenSystem.
func NewTweenSystem() *TweenSystem {
	return &TweenSystem{}
}

// Update updates all standalone tweens and sequences in the world.
func (self *TweenSystem) Update(world *World, dt float64) {
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

// FadeOverlaySystem manages fade overlays.
type FadeOverlaySystem struct{}

// NewFadeOverlaySystem creates a new FadeOverlaySystem.
func NewFadeOverlaySystem() *FadeOverlaySystem {
	return &FadeOverlaySystem{}
}

// Update updates all fade overlays in the world.
func (self *FadeOverlaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Update(dt)
	}
}

// Draw renders all fade overlays to the screen.
func (self *FadeOverlaySystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTFadeOverlay)
	for _, e := range entities {
		fadeAny, _ := world.GetComponent(e, CTFadeOverlay)
		fade := fadeAny.(*overlays.FadeOverlay)
		fade.Draw(renderer.screen)
	}
}

// CinematicOverlaySystem manages cinematic overlays.
type CinematicOverlaySystem struct{}

// NewCinematicOverlaySystem creates a new CinematicOverlaySystem.
func NewCinematicOverlaySystem() *CinematicOverlaySystem {
	return &CinematicOverlaySystem{}
}

// Update updates all cinematic overlays in the world.
func (self *CinematicOverlaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Update(dt)
	}
}

// Draw renders all cinematic overlays to the screen.
func (self *CinematicOverlaySystem) Draw(world *World, renderer *BatchRenderer) {
	entities := world.Query(CTCinematicOverlay)
	for _, e := range entities {
		cinematicAny, _ := world.GetComponent(e, CTCinematicOverlay)
		cinematic := cinematicAny.(*overlays.CinematicOverlay)
		cinematic.Draw(renderer.screen)
	}
}

// CooldownSystem manages cooldowns.
type CooldownSystem struct{}

// NewCooldownSystem creates a new CooldownSystem.
func NewCooldownSystem() *CooldownSystem {
	return &CooldownSystem{}
}

// Update advances all cooldown managers in the world by the given delta time.
func (self *CooldownSystem) Update(world *World, dt float64) {
	entities := world.Query(CTCooldown)
	for _, e := range entities {
		cmAny, _ := world.GetComponent(e, CTCooldown)
		cm := cmAny.(*managers.CooldownManager)
		cm.Update(dt)
	}
}

// DelaySystem manages delays.
type DelaySystem struct{}

// NewDelaySystem creates a new DelaySystem.
func NewDelaySystem() *DelaySystem {
	return &DelaySystem{}
}

// Update advances all delay managers in the world by the given delta time.
func (self *DelaySystem) Update(world *World, dt float64) {
	entities := world.Query(CTDelayer)
	for _, e := range entities {
		delayAny, _ := world.GetComponent(e, CTDelayer)
		delay := delayAny.(*managers.DelayManager)
		delay.Update(dt)
	}
}

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

// Define an Action type to represent custom game actions.
// This is more robust than using strings directly as it prevents typos.
type Action string

// Example List your custom game actions here.
// -----------------------------------------------------
// You can add as many as you need.
/* const (
	ActionMoveUp 	Action = "move-up"
	ActionMoveDown 	Action = "move-down"
	ActionMoveLeft 	Action = "move-left"
	ActionMoveRight Action = "move-right"
	ActionJump 		Action = "jump"
	ActionShoot 	Action = "shoot"
	ActionPause 	Action = "pause"
) */

// Define your game's key bindings.
/* var platformerBindings = map[katsu2d.Action][]katsu2d.KeyConfig{
	// Movement
	katsu2d.ActionMoveUp: 	 {{Key: ebiten.KeyW}},
	katsu2d.ActionMoveDown:  {{Key: ebiten.KeyS}},
	katsu2d.ActionMoveLeft:  {{Key: ebiten.KeyA}},
	katsu2d.ActionMoveRight: {{Key: ebiten.KeyD}},

	// Basic attack with the left mouse button
	katsu2d.ActionAttack: {
		{MouseButton: ebiten.MouseButtonLeft},
	},

	// Aiming is a continuous action, so we check if the right mouse button is held down
	katsu2d.ActionAim: {
		{MouseButton: ebiten.MouseButtonRight},
	},

	// The "advanced" action: firing an arrow
	// This happens when the left mouse button is pressed while the right mouse button is held down
	katsu2d.ActionFireArrow: {
		{
			MouseButton: ebiten.MouseButtonLeft,
			MouseButtonModifiers: []ebiten.MouseButton{ebiten.MouseButtonRight},
		},
	},
} */
// -----------------------------------------------------------------

// KeyConfig defines a single key or a combination of a key and modifiers,
// a gamepad button, or a mouse button.
type KeyConfig struct {
	Key ebiten.Key
	// Modifiers are optional keyboard keys like `ebiten.KeyShiftLeft` or `ebiten.KeyControl`.
	Modifiers []ebiten.Key
	// GamepadButton is a button on a gamepad (optional).
	GamepadButton ebiten.GamepadButton
	// GamepadModifiers are optional buttons like `ebiten.GamepadButtonRightShoulder`.
	GamepadModifiers []ebiten.GamepadButton
	// MouseButton is a button on a mouse (optional).
	MouseButton ebiten.MouseButton
	// MouseButtonModifiers are optional mouse buttons that must be held down.
	MouseButtonModifiers []ebiten.MouseButton
}

// InputSystem is an UpdateSystem that handles all game input.
type InputSystem struct {
	// A map to store the key bindings for each action.
	// An action can be bound to one or more key configurations.
	bindings map[Action][]KeyConfig

	// These maps store the current state of each action.
	// They are updated once per frame for consistent input.
	actionState   map[Action]bool
	justPressed   map[Action]bool
	justReleased  map[Action]bool
	previousState map[Action]bool
}

// NewInputSystem creates a new input system with a given set of bindings.
// This is a much more flexible approach than hardcoding them.
func NewInputSystem(bindings map[Action][]KeyConfig) *InputSystem {
	return &InputSystem{
		bindings:      bindings,
		actionState:   make(map[Action]bool),
		justPressed:   make(map[Action]bool),
		justReleased:  make(map[Action]bool),
		previousState: make(map[Action]bool),
	}
}

// Update implements the UpdateSystem interface. It polls the keyboard
// and gamepad and updates the internal state of all actions. This should be run
// once per game tick.
func (self *InputSystem) Update(world *World, dt float64) {
	// First, copy the current state to the previous state.
	for action, isPressed := range self.actionState {
		self.previousState[action] = isPressed
	}

	// Then, clear the current state to be re-evaluated.
	for action := range self.actionState {
		self.actionState[action] = false
	}

	// Iterate through all defined actions and their bindings to check for key presses.
	for action, configs := range self.bindings {
		for _, config := range configs {
			isPressed := false

			// Check for keyboard input if a key is defined.
			if config.Key != ebiten.Key(0) {
				isPressed = ebiten.IsKeyPressed(config.Key)
				// If the main key is pressed, check for modifiers.
				if isPressed && len(config.Modifiers) > 0 {
					for _, mod := range config.Modifiers {
						if !ebiten.IsKeyPressed(mod) {
							isPressed = false
							break
						}
					}
				}
			}

			// Check for gamepad input if a button is defined.
			if config.GamepadButton != ebiten.GamepadButton(0) {
				// We assume the first detected gamepad is the one being used.
				// A more advanced system could handle multiple gamepads.
				for _, gID := range ebiten.AppendGamepadIDs(nil) {
					isGamepadButtonDown := ebiten.IsGamepadButtonPressed(gID, config.GamepadButton)
					// If the main button is pressed, check for modifiers.
					if isGamepadButtonDown {
						hasAllModifiers := true
						for _, mod := range config.GamepadModifiers {
							if !ebiten.IsGamepadButtonPressed(gID, mod) {
								hasAllModifiers = false
								break
							}
						}
						if hasAllModifiers {
							isPressed = true
							break // Found a valid gamepad binding, exit this inner loop
						}
					}
				}
			}

			// Check for mouse input if a button is defined.
			if config.MouseButton != ebiten.MouseButton(0) {
				isMouseButtonDown := ebiten.IsMouseButtonPressed(config.MouseButton)
				// If the main mouse button is pressed, check for modifiers.
				if isMouseButtonDown {
					hasAllModifiers := true
					for _, mod := range config.MouseButtonModifiers {
						if !ebiten.IsMouseButtonPressed(mod) {
							hasAllModifiers = false
							break
						}
					}
					if hasAllModifiers {
						isPressed = true
						break // Found a valid mouse binding, exit this inner loop
					}
				}
			}

			// If any binding for this action is pressed, mark the action as active.
			if isPressed {
				self.actionState[action] = true
				break // Stop checking other key configs for this action.
			}
		}

		// Update the "just pressed" and "just released" states.
		// An action is "just pressed" if it's currently pressed but was not pressed last frame.
		// An action is "just released" if it's currently not pressed but was pressed last frame.
		self.justPressed[action] = self.actionState[action] && !self.previousState[action]
		self.justReleased[action] = !self.actionState[action] && self.previousState[action]
	}
}

// IsPressed returns true if the specified action is currently being held down.
// This is a fast map lookup, suitable for movement or continuous actions.
func (self *InputSystem) IsPressed(action Action) bool {
	return self.actionState[action]
}

// IsJustPressed returns true if the specified action was pressed for the first time
// in the current frame. This is ideal for single-event actions like jumping or shooting.
func (self *InputSystem) IsJustPressed(action Action) bool {
	return self.justPressed[action]
}

// IsJustReleased returns true if the specified action was released in the current frame.
// This is ideal for triggering an action when a key is no longer held down.
func (self *InputSystem) IsJustReleased(action Action) bool {
	return self.justReleased[action]
}

// ParticleEmitterSystem is an UpdateSystem that handles spawning new particles.
type ParticleEmitterSystem struct {
	tm *TextureManager
	r  *ebimath.Rand
}

// NewParticleEmitterSystem creates a new system for particle emission.
func NewParticleEmitterSystem(tm *TextureManager) *ParticleEmitterSystem {
	return &ParticleEmitterSystem{
		tm: tm,
		r:  ebimath.Random(),
	}
}

// Update implements the UpdateSystem interface to handle particle spawning.
func (self *ParticleEmitterSystem) Update(world *World, dt float64) {
	// Loop over all entities that have a Transform and an Emitter component.
	for _, entity := range world.Query(CTTransform, CTParticleEmitter) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		emitter, _ := world.GetComponent(entity, CTParticleEmitter)
		em := emitter.(*ParticleEmitterComponent)

		// Determine the number of particles to spawn this frame.
		particlesToSpawn := 0
		if em.Active {
			// Continuous emission
			em.spawnCounter += em.EmitRate * dt
			particlesToSpawn = int(em.spawnCounter)
			em.spawnCounter -= float64(particlesToSpawn)
		} else if em.BurstCount > 0 {
			// One-time burst
			particlesToSpawn = em.BurstCount
			em.BurstCount = 0 // Reset burst count after spawning
		}

		// Spawn new particles, respecting the max particle limit.
		if particlesToSpawn > 0 {
			ptEntities := world.Query(CTParticle)
			currentParticles := len(ptEntities)
			for i := 0; i < particlesToSpawn; i++ {
				if currentParticles+i >= em.MaxParticles {
					break
				}
				self.spawnParticle(world, t, em)
			}
		}
	}
}

// spawnParticle creates a single new particle entity and configures its components.
func (self *ParticleEmitterSystem) spawnParticle(world *World, emitterTransform *TransformComponent, emitter *ParticleEmitterComponent) {
	texID := 0
	if len(emitter.TextureIDs) > 1 {
		ebimath.RandomChoose(self.r, emitter.TextureIDs)
	} else {
		texID = emitter.TextureIDs[0]
	}

	tex := self.tm.Get(texID)
	width, height := tex.Bounds().Dx(), tex.Bounds().Dy()
	// Create a new entity and retrieve components from the pool for efficiency.
	newParticle := world.CreateEntity()
	particleTransform := GetTransformComponent()
	particleSprite := GetSpriteComponent()
	particleData := GetParticleComponent()

	// Particle must have the same Z with the emitter
	particleTransform.Z = emitterTransform.Z

	// Randomize particle properties based on the emitter's configuration.
	randAngle := self.r.Float64() * 2 * math.Pi // 0 to 2*pi
	randSpeed := self.r.FloatRange(
		emitter.InitialParticleSpeedMin,
		emitter.InitialParticleSpeedMax)
	randOffset := ebimath.V(self.r.Float64()*emitter.ParticleSpawnOffset.X, self.r.Float64()*emitter.ParticleSpawnOffset.Y)

	// Set initial particle state.
	particleTransform.SetPosition(emitterTransform.Position().Add(randOffset))
	particleTransform.SetRotation(ebimath.Lerp(emitter.MinRotation, emitter.MaxRotation, self.r.Float64()))
	particleTransform.SetScale(ebimath.V(
		ebimath.Lerp(emitter.MinScale, emitter.MaxScale, self.r.Float64()),
		ebimath.Lerp(emitter.MinScale, emitter.MaxScale, self.r.Float64()),
	))

	particleData.Velocity = ebimath.V(randSpeed*math.Cos(randAngle), randSpeed*math.Sin(randAngle))
	particleData.Lifetime = emitter.ParticleLifetime
	particleData.TotalLifetime = emitter.ParticleLifetime
	particleData.InitialColor = utils.GetInterpolatedColor(emitter.InitialColorMin, emitter.InitialColorMax)
	particleData.TargetColor = utils.GetInterpolatedColor(emitter.TargetColorMin, emitter.TargetColorMax)

	// Correctly setting the initial and target scale and rotation based on the emitter's ranges.
	particleData.InitialScale = particleTransform.Scale().X
	particleData.TargetScale = ebimath.Lerp(emitter.TargetScaleMin, emitter.TargetScaleMax, self.r.Float64())
	particleData.InitialRotation = particleTransform.Rotation()
	particleData.TargetRotation = ebimath.Lerp(emitter.EndRotationMin, emitter.EndRotationMax, self.r.Float64())

	particleData.Gravity = emitter.Gravity

	// Configure the sprite component.
	particleSprite.TextureID = texID
	particleSprite.Color = particleData.InitialColor
	particleSprite.SrcW = float32(width)
	particleSprite.SrcH = float32(height)

	world.AddComponent(newParticle, particleTransform)
	world.AddComponent(newParticle, particleSprite)
	world.AddComponent(newParticle, particleData)
}

// ParticleUpdateSystem is an UpdateSystem that handles the movement and lifecycle of particles.
type ParticleUpdateSystem struct {
}

// NewParticleUpdateSystem creates a new system for updating particles.
func NewParticleUpdateSystem() *ParticleUpdateSystem {
	return &ParticleUpdateSystem{}
}

// Update implements the UpdateSystem interface. It moves particles, handles fading, and removes expired particles.
func (self *ParticleUpdateSystem) Update(world *World, dt float64) {
	// A list to hold entities that should be removed at the end of the frame.
	var toRemove []Entity

	for _, entity := range world.Query(CTTransform, CTParticle, CTSprite) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		// Update position and velocity.
		p.Velocity = p.Velocity.Add(p.Gravity.MulF(dt))
		t.SetPosition(t.Position().Add(p.Velocity.MulF(dt)))
		p.Lifetime -= dt

		// Interpolate color, scale, and rotation.
		tp := 1.0 - (p.Lifetime / p.TotalLifetime) // normalized time from 0 to 1
		s.Color = utils.LerpPremultipliedRGBA(p.InitialColor, p.TargetColor, tp)
		t.SetScale(ebimath.V(
			ebimath.Lerp(p.InitialScale, p.TargetScale, tp),
			ebimath.Lerp(p.InitialScale, p.TargetScale, tp),
		))
		t.SetRotation(ebimath.Lerp(p.InitialRotation, p.TargetRotation, tp))

		if p.Lifetime <= 0 {
			toRemove = append(toRemove, entity)
		}
	}

	// Remove expired particles and return components to the pool.
	for _, entity := range toRemove {
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
		particle, _ := world.GetComponent(entity, CTParticle)
		p := particle.(*ParticleComponent)
		sprite, _ := world.GetComponent(entity, CTSprite)
		s := sprite.(*SpriteComponent)

		PutTransformComponent(t)
		PutParticleComponent(p)
		PutSpriteComponent(s) // Return the sprite component to the pool!
	}

	// Batch remove expired particles
	world.BatchRemoveEntities(toRemove...)
}

// --- Particle Render System ---

// ParticleRenderSystem is a DrawSystem that renders all particles to the screen.
type ParticleRenderSystem struct {
	tm *TextureManager
}

// NewParticleRenderSystem creates a new system for rendering particles.
// It requires a TextureManager to access particle textures.
func NewParticleRenderSystem(tm *TextureManager) *ParticleRenderSystem {
	return &ParticleRenderSystem{tm: tm}
}

// Draw implements the DrawSystem interface. It renders all particles using the BatchRenderer.
func (self *ParticleRenderSystem) Draw(world *World, renderer *BatchRenderer) {
	// Query for all entities that have a Transform and a Sprite, and are also particles.
	// This ensures we only render active particles.
	for _, entity := range world.Query(CTTransform, CTSprite, CTParticle) {
		// Use the correct component retrieval pattern
		transform, _ := world.GetComponent(entity, CTTransform)
		t := transform.(*TransformComponent)
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
