package katsu2d

import (
	"bytes"
	"fmt"
	"io"
	"math"

	"github.com/edwinsyarief/katsu2d/utils"
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

type StackingConfig struct {
	Enabled  bool
	MaxStack int
}

type StackingArray struct {
	playbackIDs []PlaybackID
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
	trackID       TrackID // New field to store the original track ID
}

type TrackData struct {
	trackID TrackID
	content []byte
	ext     string
}

// AudioManager manages all game audio, including music and sound effects.
type AudioManager struct {
	audioContext   *audio.Context
	audioBytes     map[TrackID]TrackData // Refactored to store raw audio bytes
	players        map[PlaybackID]*AudioSource
	nextPlaybackID PlaybackID
	stackingTracks map[TrackID]*StackingArray // New field
}

// NewAudioManager initializes and returns a new AudioManager.
func NewAudioManager(sampleRate int) *AudioManager {
	return &AudioManager{
		audioContext:   audio.NewContext(sampleRate),
		audioBytes:     make(map[TrackID]TrackData),
		players:        make(map[PlaybackID]*AudioSource),
		nextPlaybackID: 0,
		stackingTracks: make(map[TrackID]*StackingArray),
	}
}

// Internal helper function to decode audio from bytes
func (self *AudioManager) fromBytes(content []byte, ext string) (io.ReadSeeker, error) {
	switch ext {
	case "ogg":
		return vorbis.DecodeWithoutResampling(bytes.NewReader(content))
	case "wav":
		return wav.DecodeWithoutResampling(bytes.NewReader(content))
	case "mp3":
		return mp3.DecodeWithoutResampling(bytes.NewReader(content))
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", ext)
	}
}

// Load loads an audio file from disk and stores its bytes.
//
// NOTE: These functions assume that the `readFile`, `openEmbeddedFile`, and
// `openBundledFile` functions exist and correctly return a byte slice of the audio data.
func (self *AudioManager) Load(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("audio file path cannot be empty")
	}

	b := readFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read audio file: %s", path)
	}

	id := TrackID(len(self.audioBytes))
	self.audioBytes[id] = TrackData{trackID: id, content: b, ext: utils.GetFileExtension(path)}
	return id, nil
}

// LoadEmbedded loads embedded audio data and stores its bytes.
func (self *AudioManager) LoadEmbedded(path string) (TrackID, error) {
	b := openEmbeddedFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read embedded audio file: %s", path)
	}

	id := TrackID(len(self.audioBytes))
	self.audioBytes[id] = TrackData{trackID: id, content: b, ext: utils.GetFileExtension(path)}
	return id, nil
}

// LoadFromAssetPacker loads audio from an asset pack and stores its bytes.
func (self *AudioManager) LoadFromAssetPacker(path string) (TrackID, error) {
	b := openBundledFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read bundled audio file: %s", path)
	}

	id := TrackID(len(self.audioBytes))
	self.audioBytes[id] = TrackData{trackID: id, content: b, ext: utils.GetFileExtension(path)}
	return id, nil
}

// Helper method to prepare a new audio source from stored bytes
func (self *AudioManager) prepareAudioSource(trackID TrackID, pan float64, loop bool) (*AudioSource, error) {
	if int(trackID) < 0 || int(trackID) >= len(self.audioBytes) {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}

	audioBytes := self.audioBytes[trackID]
	reader, err := self.fromBytes(audioBytes.content, audioBytes.ext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio from bytes: %w", err)
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
		trackID:   trackID, // Set the trackID here
	}, nil
}

// PlaySound plays a one-shot sound effect.
func (self *AudioManager) PlaySound(trackID TrackID, pan float64, stackConfig *StackingConfig) (PlaybackID, error) {
	if stackConfig != nil && stackConfig.Enabled {
		return self.playStackedSound(trackID, pan, *stackConfig)
	}

	// Stop existing instances if not stacking
	self.StopByTrackID(trackID)

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

func (self *AudioManager) playStackedSound(trackID TrackID, pan float64, stackConfig StackingConfig) (PlaybackID, error) {
	stackArray, exists := self.stackingTracks[trackID]
	if !exists {
		stackArray = &StackingArray{
			playbackIDs: make([]PlaybackID, 0),
		}
		self.stackingTracks[trackID] = stackArray
	}

	// Handle stack limit
	if len(stackArray.playbackIDs) >= stackConfig.MaxStack {
		oldestID := stackArray.playbackIDs[0]
		_ = self.Stop(oldestID)
		stackArray.playbackIDs = stackArray.playbackIDs[1:]
	}

	source, err := self.prepareAudioSource(trackID, pan, false)
	if err != nil {
		return -1, err
	}

	source.currentVolume = 1.0
	source.player.Play()

	playbackID := self.nextPlaybackID
	self.players[playbackID] = source
	self.nextPlaybackID++

	stackArray.playbackIDs = append(stackArray.playbackIDs, playbackID)

	return playbackID, nil
}

// PlayMusic plays a music track.
func (self *AudioManager) PlayMusic(trackID TrackID, loop bool) (PlaybackID, error) {
	// Stop any existing instances of this specific music track
	self.StopByTrackID(trackID)

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
// The allowStacking parameter controls if multiple instances of the same sound can play.
func (self *AudioManager) FadeSound(trackID TrackID, pan, fadeDuration float64, fadeType AudioFadeType, stackConfig *StackingConfig) (PlaybackID, error) {
	if stackConfig != nil && stackConfig.Enabled {
		return self.fadeStackedSound(trackID, pan, fadeDuration, fadeType, *stackConfig)
	}

	self.StopByTrackID(trackID)
	return self.initFade(trackID, pan, false, fadeDuration, fadeType, 1.0)
}

func (self *AudioManager) fadeStackedSound(trackID TrackID, pan, fadeDuration float64, fadeType AudioFadeType, stackConfig StackingConfig) (PlaybackID, error) {
	stackArray, exists := self.stackingTracks[trackID]
	if !exists {
		stackArray = &StackingArray{
			playbackIDs: make([]PlaybackID, 0),
		}
		self.stackingTracks[trackID] = stackArray
	}

	if len(stackArray.playbackIDs) >= stackConfig.MaxStack {
		oldestID := stackArray.playbackIDs[0]
		self.Stop(oldestID)
		stackArray.playbackIDs = stackArray.playbackIDs[1:]
	}

	playbackID, err := self.initFade(trackID, pan, false, fadeDuration, fadeType, 1.0)
	if err != nil {
		return -1, err
	}

	stackArray.playbackIDs = append(stackArray.playbackIDs, playbackID)

	return playbackID, nil
}

// FadeMusic plays a music track with a fade effect.
// Music is typically not stacked, so this will still stop any existing instances.
func (self *AudioManager) FadeMusic(trackID TrackID, loop bool, fadeDuration float64, fadeType AudioFadeType) (PlaybackID, error) {
	// Stop any existing instances of this specific music track
	self.StopByTrackID(trackID)
	return self.initFade(trackID, 0, loop, fadeDuration, fadeType, 1.0)
}

// Stop stops and removes a single playing audio source.
func (self *AudioManager) Stop(id PlaybackID) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}

	// Remove from stacking array if present
	if stackArray, exists := self.stackingTracks[source.trackID]; exists {
		for i, stackedID := range stackArray.playbackIDs {
			if stackedID == id {
				stackArray.playbackIDs = append(stackArray.playbackIDs[:i], stackArray.playbackIDs[i+1:]...)
				break
			}
		}
		// Clean up empty stacking arrays
		if len(stackArray.playbackIDs) == 0 {
			delete(self.stackingTracks, source.trackID)
		}
	}

	source.player.Pause()
	source.player.Close()
	delete(self.players, id)
	return nil
}

// StopByTrackID stops all playing instances of a specific track.
func (self *AudioManager) StopByTrackID(trackID TrackID) {
	// Create a slice to hold the playback IDs to stop
	var idsToStop []PlaybackID
	for id, source := range self.players {
		if source.trackID == trackID {
			idsToStop = append(idsToStop, id)
		}
	}
	// Stop each player found
	for _, id := range idsToStop {
		_ = self.Stop(id)
	}
}

// Pause pauses a single playing audio source, allowing it to be resumed later.
func (self *AudioManager) Pause(id PlaybackID) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.player.Pause()
	return nil
}

// Resume resumes a paused audio source.
func (self *AudioManager) Resume(id PlaybackID) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.player.Play()
	return nil
}

// IsPlaying checks if a specific playback instance is currently playing.
func (self *AudioManager) IsPlaying(id PlaybackID) bool {
	source, ok := self.players[id]
	if !ok {
		return false
	}
	return source.player.IsPlaying()
}

// IsTrackPlaying checks if any instance of a given track ID is currently playing.
func (self *AudioManager) IsTrackPlaying(trackID TrackID) bool {
	for _, source := range self.players {
		if source.trackID == trackID && source.player.IsPlaying() {
			return true
		}
	}
	return false
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
			self.Stop(id)
		}
	}
}
