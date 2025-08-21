package main

import (
	_ "image/png"
	"log"
	"math/rand"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/edwinsyarief/katsu2d"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Game implements ebiten.Game interface.
type Game struct {
	engine *katsu2d.Engine
	img    *ebiten.Image
}

// NewGame creates a new Game object and sets up the engine.
func NewGame() *Game {
	g := &Game{}

	g.engine = katsu2d.NewEngine(
		katsu2d.WithWindowSize(640, 480),
		katsu2d.WithWindowTitle("Foliage Example"),
	)

	tm := g.engine.TextureManager()
	world := g.engine.World()

	// --- Texture Loading ---
	foliageImage, _, _ := ebitenutil.NewImageFromFile("./examples/foliage/grass.png")
	g.img = foliageImage
	foliageTextureID := tm.Add(foliageImage)

	// --- System Setup ---
	g.engine.AddUpdateSystem(katsu2d.NewFoliageSystem())
	g.engine.AddBackgroundDrawSystem(katsu2d.NewSpriteRenderSystem(world, tm))

	// --- Foliage Controller ---
	foliageControllerEntity := world.CreateEntity()
	foliageController := katsu2d.NewFoliageControllerComponent(
		katsu2d.WithFoliageWindForce(0.3), // Radians
		katsu2d.WithFoliageWindSpeed(2.5),
	)
	world.AddComponent(foliageControllerEntity, foliageController)

	// --- Foliage Entities ---
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 120, ebimath.V(0.5, 1.0))
	}
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 140, ebimath.V(0.5, 1.0))
	}
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 160, ebimath.V(0.5, 1.0))
	}
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 180, ebimath.V(0.5, 1.0))
	}
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 200, ebimath.V(0.5, 1.0))
	}
	for i := 0; i < 10; i++ {
		createFoliage(world, tm, foliageTextureID, float64(i*60), 220, ebimath.V(0.5, 1.0))
	}

	return g
}

func createFoliage(world *katsu2d.World, tm *katsu2d.TextureManager, textureID int, x, y float64, pivot ebimath.Vector) {
	entity := world.CreateEntity()

	// Transform
	transform := katsu2d.NewTransformComponent()
	transform.SetPosition(ebimath.V(x, y))
	transform.SetOffset(ebimath.V(20, 0))
	world.AddComponent(entity, transform)

	// Sprite with a grid mesh
	img := tm.Get(textureID)
	sprite := katsu2d.NewSpriteComponent(textureID, img.Bounds())
	sprite.DstW = 128
	sprite.DstH = 128
	sprite.SetGrid(5, 3) // 1 column, 10 rows
	world.AddComponent(entity, sprite)

	// Foliage
	foliage := &katsu2d.FoliageComponent{
		TextureID: textureID,
		SwaySeed:  rand.Float64() * 100,
		Pivot:     pivot,
	}
	world.AddComponent(entity, foliage)
}

func (self *Game) Update() error {
	return self.engine.Update()
}

func (self *Game) Draw(screen *ebiten.Image) {
	self.engine.Draw(screen)
	screen.DrawImage(self.img, nil)
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
