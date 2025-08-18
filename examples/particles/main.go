package main

import (
	"log"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game implements ebiten.Game interface.
type Game struct {
	engine *katsu2d.Engine
}

// NewGame creates a new Game object and sets up the engine.
func NewGame() *Game {
	g := &Game{}

	// --- Engine Setup ---
	g.engine = katsu2d.NewEngine(
		katsu2d.WithWindowSize(1280, 720),
		katsu2d.WithWindowTitle("Particle Example"),
	)

	tm := g.engine.TextureManager()
	world := g.engine.World()

	// --- Player Entity Setup ---
	playerEntity := world.CreateEntity()
	playerTransform := katsu2d.NewTransformComponent()
	// Start the emitter at the bottom-center of the screen
	playerTransform.SetPosition(ebimath.V(320, 200))
	world.AddComponent(playerEntity, playerTransform)

	// --- Particle Emitter Setup ---
	firePreset := katsu2d.FirePreset(0)
	world.AddComponent(playerEntity, firePreset)

	// --- System Setup ---
	g.engine.AddUpdateSystem(katsu2d.NewParticleEmitterSystem(tm))
	g.engine.AddUpdateSystem(katsu2d.NewParticleUpdateSystem())
	g.engine.AddBackgroundDrawSystem(katsu2d.NewParticleRenderSystem(tm))

	return g
}

func (self *Game) Update() error {
	return self.engine.Update()
}

func (self *Game) Draw(screen *ebiten.Image) {
	self.engine.Draw(screen)
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
