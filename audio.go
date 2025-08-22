package katsu2d

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// TrackID is a unique identifier for a loaded audio track.
type TrackID int

// PlaybackID is a unique identifier for a single playing instance of an audio track.
type PlaybackID int

// FadeType defines the direction of an audio fade.
type AudioFadeType int

const (
	// FadeIn indicates the volume will increase.
	FadeIn AudioFadeType = iota
	// FadeOut indicates the volume will decrease.
	FadeOut
)

// StereoPanStream is an audio buffer that changes the stereo channel's signal
// based on the Panning.
type StereoPanStream struct {
	io.ReadSeeker
	pan float64 // -1: left; 0: center; 1: right
	buf []byte
}

// Read reads panned audio data into p.
func (self *StereoPanStream) Read(p []byte) (int, error) {
	// If the stream has a buffer that was read in the previous time, use this first.
	var bufN int
	if len(self.buf) > 0 {
		bufN = copy(p, self.buf)
		self.buf = self.buf[bufN:]
	}

	readN, err := self.ReadSeeker.Read(p[bufN:])
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Align the buffer size in multiples of 8 (4 bytes per channel, 2 channels).
	totalN := bufN + readN
	extra := totalN % 8
	self.buf = append(self.buf, p[totalN-extra:totalN]...)
	alignedN := totalN - extra

	// Calculate stereo balance using Unity's approach
	ls := float32(math.Min(self.pan*-1+1, 1))
	rs := float32(math.Min(self.pan+1, 1))

	for i := 0; i < alignedN; i += 8 {
		lc := math.Float32frombits(uint32(p[i])|(uint32(p[i+1])<<8)|(uint32(p[i+2])<<16)|(uint32(p[i+3])<<24)) * ls
		rc := math.Float32frombits(uint32(p[i+4])|(uint32(p[i+5])<<8)|(uint32(p[i+6])<<16)|(uint32(p[i+7])<<24)) * rs

		lcBits := math.Float32bits(lc)
		rcBits := math.Float32bits(rc)

		p[i] = byte(lcBits)
		p[i+1] = byte(lcBits >> 8)
		p[i+2] = byte(lcBits >> 16)
		p[i+3] = byte(lcBits >> 24)
		p[i+4] = byte(rcBits)
		p[i+5] = byte(rcBits >> 8)
		p[i+6] = byte(rcBits >> 16)
		p[i+7] = byte(rcBits >> 24)
	}
	return alignedN, err
}

// SetPan sets the pan value for the stream, clamped between -1 and 1.
func (self *StereoPanStream) SetPan(pan float64) {
	self.pan = math.Min(math.Max(-1, pan), 1)
}

// Pan returns the current pan value of the stream.
func (self *StereoPanStream) Pan() float64 {
	return self.pan
}

// NewStereoPanStream returns a new StereoPanStream with a buffered src.
func NewStereoPanStream(src io.ReadSeeker) *StereoPanStream {
	return &StereoPanStream{
		ReadSeeker: src,
	}
}

// InfiniteLoop wraps an io.ReadSeeker to loop indefinitely.
type InfiniteLoop struct {
	track io.ReadSeeker
}

func (self *InfiniteLoop) Read(p []byte) (int, error) {
	n, err := self.track.Read(p)
	if err == io.EOF {
		if _, err := self.track.Seek(0, io.SeekStart); err != nil {
			return n, err
		}
		n, err = self.track.Read(p)
	}
	return n, err
}

func (self *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	return self.track.Seek(offset, whence)
}

// AudioSource represents a single playing instance of a sound or music track.
type AudioSource struct {
	player        *audio.Player
	panStream     *StereoPanStream
	isFading      bool
	fadeDuration  float64
	currentVolume float64
	targetVolume  float64
	fadeType      AudioFadeType
}

// AudioManager manages all game audio, including music and sound effects.
type AudioManager struct {
	audioContext   *audio.Context
	readers        []io.ReadSeeker
	players        map[PlaybackID]*AudioSource
	nextPlaybackID PlaybackID
}

// NewAudioManager initializes and returns a new AudioManager.
func NewAudioManager(sampleRate int) *AudioManager {
	return &AudioManager{
		audioContext:   audio.NewContext(sampleRate),
		readers:        make([]io.ReadSeeker, 0),
		players:        make(map[PlaybackID]*AudioSource),
		nextPlaybackID: 0,
	}
}

// Internal helper function to decode audio from bytes
func (self *AudioManager) fromBytes(content []byte, ext string) (io.ReadSeeker, error) {
	switch ext {
	case ".ogg":
		return vorbis.DecodeF32(bytes.NewReader(content))
	case ".wav":
		return wav.DecodeF32(bytes.NewReader(content))
	case ".mp3":
		return mp3.DecodeF32(bytes.NewReader(content))
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", ext)
	}
}

// Load methods (keeping your existing implementation)
func (self *AudioManager) Load(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("audio file path cannot be empty")
	}

	ext := path[len(path)-4:]
	b := readFile(path)
	s, err := self.fromBytes(b, ext)
	if err != nil {
		return -1, err
	}

	id := TrackID(len(self.readers))
	self.readers = append(self.readers, s)
	return id, nil
}

func (self *AudioManager) LoadEmbedded(path string) (TrackID, error) {
	b := openEmbeddedFile(path)
	ext := path[len(path)-4:]
	s, err := self.fromBytes(b, ext)
	if err != nil {
		return -1, err
	}

	id := TrackID(len(self.readers))
	self.readers = append(self.readers, s)
	return id, nil
}

func (self *AudioManager) LoadFromAssetPacker(path string) (TrackID, error) {
	b := openBundledFile(path)
	ext := path[len(path)-4:]
	s, err := self.fromBytes(b, ext)
	if err != nil {
		return -1, err
	}

	id := TrackID(len(self.readers))
	self.readers = append(self.readers, s)
	return id, nil
}

// Helper method to prepare an audio source
func (self *AudioManager) prepareAudioSource(trackID TrackID, pan float64, loop bool) (*AudioSource, error) {
	if int(trackID) < 0 || int(trackID) >= len(self.readers) {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}

	reader := self.readers[trackID]
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek audio: %w", err)
	}

	var stream io.ReadSeeker = reader
	if loop {
		stream = &InfiniteLoop{track: reader}
	}

	panStream := NewStereoPanStream(stream)
	if pan != 0 {
		panStream.SetPan(pan)
	}

	player, err := self.audioContext.NewPlayer(panStream)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio player: %w", err)
	}

	return &AudioSource{
		player:    player,
		panStream: panStream,
	}, nil
}

// PlaySound plays a one-shot sound effect.
func (self *AudioManager) PlaySound(trackID TrackID, pan float64) (PlaybackID, error) {
	source, err := self.prepareAudioSource(trackID, pan, false)
	if err != nil {
		return -1, err
	}

	source.currentVolume = 1.0
	source.player.Play()

	playbackID := self.nextPlaybackID
	self.players[playbackID] = source
	self.nextPlaybackID++

	return playbackID, nil
}

// PlayMusic plays a music track.
func (self *AudioManager) PlayMusic(trackID TrackID, loop bool) (PlaybackID, error) {
	source, err := self.prepareAudioSource(trackID, 0, loop)
	if err != nil {
		return -1, err
	}

	const defaultMusicVolume = 0.5
	source.currentVolume = defaultMusicVolume
	source.player.SetVolume(defaultMusicVolume)
	source.player.Play()

	playbackID := self.nextPlaybackID
	self.players[playbackID] = source
	self.nextPlaybackID++

	return playbackID, nil
}

// Helper method for fade initialization
func (self *AudioManager) initFade(trackID TrackID, pan float64, loop bool, fadeDuration float64,
	fadeType AudioFadeType, defaultVolume float64) (PlaybackID, error) {

	source, err := self.prepareAudioSource(trackID, pan, loop)
	if err != nil {
		return -1, err
	}

	if fadeDuration <= 0 {
		return -1, fmt.Errorf("fade duration must be greater than 0")
	}

	startVolume := defaultVolume
	targetVolume := defaultVolume
	if fadeType == FadeIn {
		startVolume = 0
		source.player.SetVolume(0)
	} else {
		targetVolume = 0
	}

	source.player.Play()
	source.isFading = true
	source.fadeDuration = fadeDuration
	source.currentVolume = startVolume
	source.targetVolume = targetVolume
	source.fadeType = fadeType

	playbackID := self.nextPlaybackID
	self.players[playbackID] = source
	self.nextPlaybackID++

	return playbackID, nil
}

// FadeSound plays a sound effect with a fade effect.
func (self *AudioManager) FadeSound(trackID TrackID, pan, fadeDuration float64, fadeType AudioFadeType) (PlaybackID, error) {
	return self.initFade(trackID, pan, false, fadeDuration, fadeType, 1.0)
}

// FadeMusic plays a music track with a fade effect.
func (self *AudioManager) FadeMusic(trackID TrackID, loop bool, fadeDuration float64, fadeType AudioFadeType) (PlaybackID, error) {
	return self.initFade(trackID, 0, loop, fadeDuration, fadeType, 0.5)
}

// Stop stops and removes a single playing audio source.
func (self *AudioManager) Stop(id PlaybackID) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.player.Pause()
	source.player.Close()
	delete(self.players, id)
	return nil
}

// StopAll stops all currently playing audio sources.
func (self *AudioManager) StopAll() {
	for id := range self.players {
		_ = self.Stop(id)
	}
}

// SetVolume adjusts the volume of a single playing audio source.
func (self *AudioManager) SetVolume(id PlaybackID, volume float64) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.currentVolume = volume
	source.player.SetVolume(volume)
	return nil
}

// SetPan adjusts the stereo pan of a single playing audio source.
func (self *AudioManager) SetPan(id PlaybackID, pan float64) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.panStream.SetPan(pan)
	return nil
}

// Update handles audio state updates and cleanup.
func (self *AudioManager) Update(dt float64) {
	for id, source := range self.players {
		if source.isFading {
			// Calculate volume change per frame
			volumeChange := (1.0 / source.fadeDuration) * dt
			if source.fadeType == FadeOut {
				volumeChange = -volumeChange
			}

			source.currentVolume += volumeChange
			fadeComplete := (source.fadeType == FadeIn && source.currentVolume >= source.targetVolume) ||
				(source.fadeType == FadeOut && source.currentVolume <= source.targetVolume)

			if fadeComplete {
				source.currentVolume = source.targetVolume
				source.isFading = false
			}

			source.player.SetVolume(source.currentVolume)
		}

		// Clean up finished players
		if !source.player.IsPlaying() && !source.isFading {
			source.player.Close()
			delete(self.players, id)
		}
	}
}
