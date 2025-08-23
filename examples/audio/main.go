package main

import (
	"log"

	"github.com/edwinsyarief/katsu2d"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ActionPlaySound katsu2d.Action = "play_sound"
)

var keybindings = map[katsu2d.Action][]katsu2d.KeyConfig{
	ActionPlaySound: {{Key: ebiten.KeySpace}},
}

// AudioSystem is a simple system to play a sound when the spacebar is pressed.
type AudioSystem struct {
	audioManager *katsu2d.AudioManager
	trackID      katsu2d.TrackID
	playbackID   katsu2d.PlaybackID
	stackConfig  *katsu2d.StackingConfig
}

func (s *AudioSystem) Update(world *katsu2d.World, dt float64) {
	for _, e := range world.Query(katsu2d.CTInput) {
		i, _ := world.GetComponent(e, katsu2d.CTInput)
		input := i.(*katsu2d.InputComponent)

		if input.IsJustPressed(ActionPlaySound) {
			s.playbackID, _ = s.audioManager.FadeMusic(s.trackID, true, 3.75, katsu2d.FadeIn)
		}
	}
}

// Game implements ebiten.Game interface.
type Game struct {
	engine       *katsu2d.Engine
	audioManager *katsu2d.AudioManager
}

// NewGame creates a new Game object and sets up the engine.
func NewGame() *Game {
	g := &Game{}

	// --- Engine Setup ---
	g.engine = katsu2d.NewEngine(
		katsu2d.WithWindowSize(320, 240),
		katsu2d.WithWindowTitle("Audio Example"),
		katsu2d.WithUpdateSystem(katsu2d.NewInputSystem()),
	)

	world := g.engine.World()

	// --- Audio Setup ---
	g.audioManager = g.engine.AudioManager()
	trackID, err := g.audioManager.Load("./examples/audio/piano.mp3")
	if err != nil {
		log.Fatalf("failed to load audio file: %v", err)
	}

	// --- Entity Setup ---
	// Create an entity that will handle input
	inputEntity := world.CreateEntity()
	world.AddComponent(inputEntity, katsu2d.NewInputComponent(keybindings))

	// --- System Setup ---
	g.engine.AddUpdateSystem(&AudioSystem{
		audioManager: g.audioManager,
		trackID:      trackID,
		stackConfig: &katsu2d.StackingConfig{
			Enabled:  true,
			MaxStack: 3, // Allow up to 3 instances of the same sound
		},
	})

	return g
}

func (g *Game) Update() error {
	if err := g.engine.Update(); err != nil {
		return err
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.engine.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return g.engine.Layout(outsideWidth, outsideHeight)
}

func main() {
	game := NewGame()
	if err := game.engine.Run(); err != nil {
		log.Fatal(err)
	}
}
