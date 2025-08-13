package katsu2d

import (
	"bytes"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// AudioManager manages audio.
type AudioManager struct {
	context *audio.Context
	sounds  map[string][]byte // assumes mp3
}

// NewAudioManager creates a new audio manager.
func NewAudioManager(sampleRate int) *AudioManager {
	return &AudioManager{
		context: audio.NewContext(sampleRate),
		sounds:  make(map[string][]byte),
	}
}

// LoadMP3 loads an MP3 sound from bytes.
func (am *AudioManager) LoadMP3(name string, data []byte) {
	am.sounds[name] = data
}

// Play plays a sound by name.
func (am *AudioManager) Play(name string) (*audio.Player, error) {
	data, ok := am.sounds[name]
	if !ok {
		return nil, fmt.Errorf("sound not found: %s", name)
	}
	stream, err := mp3.DecodeWithSampleRate(am.context.SampleRate(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	player, err := am.context.NewPlayer(stream)
	if err != nil {
		return nil, err
	}
	player.Play()
	return player, nil
}
