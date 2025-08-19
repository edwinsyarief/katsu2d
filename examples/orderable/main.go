package main

import (
	"fmt"
	"image/color"
	"log"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ActionMoveUp    katsu2d.Action = "move_up"
	ActionMoveDown  katsu2d.Action = "move_down"
	ActionMoveLeft  katsu2d.Action = "move_left"
	ActionMoveRight katsu2d.Action = "move_right"
)

var keybindings = map[katsu2d.Action][]katsu2d.KeyConfig{
	ActionMoveUp:    {{Key: ebiten.KeyW}, {Key: ebiten.KeyUp}},
	ActionMoveDown:  {{Key: ebiten.KeyS}, {Key: ebiten.KeyDown}},
	ActionMoveLeft:  {{Key: ebiten.KeyA}, {Key: ebiten.KeyLeft}},
	ActionMoveRight: {{Key: ebiten.KeyD}, {Key: ebiten.KeyRight}},
}

const PlayerTag = "player"

// PlayerSystem is a simple system to move the player.
type PlayerSystem struct{}

func (self *PlayerSystem) Update(world *katsu2d.World, dt float64) {
	var player katsu2d.Entity
	found := false
	for _, e := range world.Query(katsu2d.CTTag, katsu2d.CTTransform, katsu2d.CTInput) {
		tag, _ := world.GetComponent(e, katsu2d.CTTag)
		if tag.(*katsu2d.TagComponent).Tag == PlayerTag {
			player = e
			found = true
			break
		}
	}

	if !found {
		return
	}

	t, _ := world.GetComponent(player, katsu2d.CTTransform)
	transform := t.(*katsu2d.TransformComponent)

	i, _ := world.GetComponent(player, katsu2d.CTInput)
	input := i.(*katsu2d.InputComponent)

	speed := 60.0
	var velocity ebimath.Vector
	if input.IsPressed(ActionMoveUp) {
		velocity.Y = -1
	}
	if input.IsPressed(ActionMoveDown) {
		velocity.Y = 1
	}
	if input.IsPressed(ActionMoveLeft) {
		velocity.X = -1
	}
	if input.IsPressed(ActionMoveRight) {
		velocity.X = 1
	}

	if !velocity.IsZero() {
		newPos := transform.Position().Add(velocity.Normalized().MulF(speed * dt))
		transform.SetPosition(newPos)
	}
}

// Game implements ebiten.Game interface.
type Game struct {
	engine *katsu2d.Engine
}

// NewGame creates a new Game object and sets up the engine.
func NewGame() *Game {
	g := &Game{}

	g.engine = katsu2d.NewEngine(
		katsu2d.WithWindowSize(600, 480),
		katsu2d.WithWindowTitle("Render Order Example"),
		katsu2d.WithClearScreenEachFrame(false),
	)

	tm := g.engine.TextureManager()
	world := g.engine.World()

	// --- Texture Loading ---
	treeImg := ebiten.NewImage(25, 50)
	treeImg.Fill(color.RGBA{R: 93, G: 62, B: 4, A: 255})
	treeTexID := tm.Add(treeImg)

	playerImg := ebiten.NewImage(25, 25)
	playerImg.Fill(color.White)
	playerTexID := tm.Add(playerImg)

	particleImg := ebiten.NewImage(6, 6)
	particleImg.Fill(color.RGBA{R: 255, G: 0, B: 0, A: 255})
	particleTexID := tm.Add(particleImg)

	// --- Entities ---
	// Create some trees
	for i := 0; i < 100; i++ {
		treeEntity := world.CreateEntity()
		treeTransform := katsu2d.NewTransformComponent()
		treeTransform.SetPosition(ebimath.V(float64(i*40+20), 100))
		world.AddComponent(treeEntity, treeTransform)
		treeSprite := katsu2d.NewSpriteComponent(treeTexID, treeImg.Bounds())
		world.AddComponent(treeEntity, treeSprite)
		world.AddComponent(treeEntity, katsu2d.NewOrderableComponent(func() float64 {
			return treeTransform.Position().Y + float64(treeSprite.DstH)
		}))
	}
	for i := 0; i < 100; i++ {
		treeEntity := world.CreateEntity()
		treeTransform := katsu2d.NewTransformComponent()
		treeTransform.SetPosition(ebimath.V(float64(i*40+40), 200))
		world.AddComponent(treeEntity, treeTransform)
		treeSprite := katsu2d.NewSpriteComponent(treeTexID, treeImg.Bounds())
		world.AddComponent(treeEntity, treeSprite)
		world.AddComponent(treeEntity, katsu2d.NewOrderableComponent(func() float64 {
			return treeTransform.Position().Y + float64(treeSprite.DstH)
		}))
	}

	// Player
	playerEntity := world.CreateEntity()
	playerTransform := katsu2d.NewTransformComponent()
	playerTransform.SetPosition(ebimath.V(160, 120))
	world.AddComponent(playerEntity, playerTransform)
	playerSprite := katsu2d.NewSpriteComponent(playerTexID, playerImg.Bounds())
	world.AddComponent(playerEntity, playerSprite)
	world.AddComponent(playerEntity, katsu2d.NewTagComponent(PlayerTag))
	world.AddComponent(playerEntity, katsu2d.NewInputComponent(keybindings))
	world.AddComponent(playerEntity, katsu2d.NewOrderableComponent(func() float64 {
		return playerTransform.Position().Y + float64(playerSprite.DstH)
	}))

	// Particle Emitter
	firePreset := katsu2d.FirePreset(particleTexID)
	world.AddComponent(playerEntity, firePreset)

	// --- Systems ---
	g.engine.AddUpdateSystem(katsu2d.NewInputSystem())
	g.engine.AddUpdateSystem(&PlayerSystem{})
	g.engine.AddUpdateSystem(katsu2d.NewParticleEmitterSystem(tm))
	g.engine.AddUpdateSystem(katsu2d.NewParticleUpdateSystem())

	renderOrderSystem := katsu2d.NewOrderableSystem(world, tm)
	g.engine.AddUpdateSystem(renderOrderSystem)
	g.engine.AddBackgroundDrawSystem(renderOrderSystem)

	return g
}

func (self *Game) Update() error {
	return self.engine.Update()
}

func (self *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)
	self.engine.Draw(screen)
	println(fmt.Sprintf("FPS: %v", ebiten.ActualFPS()))
}

func (self *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return self.engine.Layout(outsideWidth, outsideHeight)
}

func main() {
	game := NewGame()
	if err := game.engine.Run(); err != nil {
		log.Fatal(err)
	}
}
