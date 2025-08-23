package main

import (
	"log"

	"github.com/edwinsyarief/katsu2d"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	ActionPlayMusic katsu2d.Action = "play_music"
	ActionPlaySfx   katsu2d.Action = "play_sfx"
)

var keybindings = map[katsu2d.Action][]katsu2d.KeyConfig{
	ActionPlayMusic: {{Key: ebiten.KeySpace}},
	ActionPlaySfx:   {{Key: ebiten.KeyS}},
}

// AudioSystem is a simple system to play a sound when the spacebar is pressed.
type AudioSystem struct {
	audioManager  *katsu2d.AudioManager
	trackID       katsu2d.TrackID
	playbackID    katsu2d.PlaybackID
	stackConfig   *katsu2d.StackingConfig
	sfxID         katsu2d.TrackID
	sfxPlaybackID katsu2d.PlaybackID
}

func (s *AudioSystem) Update(world *katsu2d.World, dt float64) {
	for _, e := range world.Query(katsu2d.CTInput) {
		i, _ := world.GetComponent(e, katsu2d.CTInput)
		input := i.(*katsu2d.InputComponent)

		if input.IsJustPressed(ActionPlayMusic) {
			s.playbackID, _ = s.audioManager.FadeSound(s.trackID, 0, 3.75, katsu2d.AudioFadeIn, s.stackConfig)
		}
		if input.IsJustPressed(ActionPlaySfx) {
			s.sfxPlaybackID, _ = s.audioManager.PlaySound(s.sfxID, 0, s.stackConfig)
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
	trackID, err := g.audioManager.Load("./examples/audio/piano.ogg")
	if err != nil {
		log.Fatalf("failed to load audio file: %v", err)
	}
	sfxID, err := g.audioManager.Load("./examples/audio/wood-step.mp3")
	if err != nil {
		log.Fatalf("failed to load sfx file: %v", err)
	}

	// --- Entity Setup ---
	// Create an entity that will handle input
	inputEntity := world.CreateEntity()
	world.AddComponent(inputEntity, katsu2d.NewInputComponent(keybindings))

	// --- System Setup ---
	g.engine.AddUpdateSystem(&AudioSystem{
		audioManager: g.audioManager,
		trackID:      trackID,
		sfxID:        sfxID,
		stackConfig: &katsu2d.StackingConfig{
			Enabled:  true,
			MaxStack: 1, // Allow up to 3 instances of the same sound
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
