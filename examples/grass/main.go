package main

import (
	"image/color"
	"log"

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
		katsu2d.WithWindowSize(640, 480),
		katsu2d.WithWindowTitle("Grass Example"),
		katsu2d.WithUpdateSystem(katsu2d.NewGrassControllerSystem()),
	)

	tm := g.engine.TextureManager()
	world := g.engine.World()

	// --- Texture Loading ---
	// Grass
	grass := ebiten.NewImage(8, 32)
	grass.Fill(color.RGBA{R: 58, G: 93, B: 35, A: 255})
	texId := tm.Add(grass) // ID 1: "grass"

	// grass controller
	transform := katsu2d.NewTransformComponent()
	grassController := katsu2d.NewGrassControllerComponent(world, tm,
		640, 480, texId, transform.Z,
		katsu2d.WithGrassDensity(5),
		katsu2d.WithGrassNoiseMapSize(512),
		katsu2d.WithGrassNoiseFrequency(30),
		katsu2d.WithGrassWindDirection(1, 0),
		katsu2d.WithGrassAreas([]katsu2d.Area{
			{X1: 1, Y1: 1, X2: 10, Y2: 10},
		}),
	)
	entity := world.CreateEntity()
	world.AddComponent(entity, grassController)

	// 1. TileMapRenderSystem draws the background (lower grid).
	g.engine.AddBackgroundDrawSystem(katsu2d.NewSpriteRenderSystem(world, tm))

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
