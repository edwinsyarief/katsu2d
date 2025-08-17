package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/edwinsyarief/katsu2d"
	"github.com/edwinsyarief/katsu2d/grass"
)

type Game struct {
	engine *katsu2d.Engine
	world  *katsu2d.World
}

func (g *Game) Update() error {
	g.engine.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.engine.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 800, 600
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Grass Test")

	grass.RegisterComponents()

	engine := katsu2d.NewEngine()

	world := engine.World()

	grassImage := ebiten.NewImage(16, 16)
	grassImage.Fill(color.RGBA{R: 0, G: 255, B: 0, A: 255})
	tm := engine.TextureManager()
	textureID := tm.Add(grassImage)

	grassController := grass.NewGrassControllerComponent(world, tm, 800, 600, textureID, 0)
	grassControllerEntity := world.CreateEntity()
	world.AddComponent(grassControllerEntity, grassController)
	engine.AddUpdateSystem(&grass.GrassControllerSystem{})

	cameraEntity := world.CreateEntity()
	cameraComponent := katsu2d.NewCameraComponent(800, 600)
	world.AddComponent(cameraEntity, cameraComponent)
	engine.AddUpdateSystem(&katsu2d.CameraSystem{})

	game := &Game{
		engine: engine,
		world:  world,
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
