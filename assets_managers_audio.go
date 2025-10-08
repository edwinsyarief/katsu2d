package katsu2d

import (
	"bytes"
	"fmt"
	"io"
	"log"
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
	AudioFadeIn AudioFadeType = iota
	// FadeOut indicates the volume will decrease.
	AudioFadeOut
)

// StereoPanStream is an audio buffer that changes the stereo channel's signal
// based on the Panning.
type StereoPanStream struct {
	io.ReadSeeker
	buf []byte
	pan float64 // -1: left; 0: center; 1: right
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

// StackingConfig defines how multiple instances of the same sound are handled.
type StackingConfig struct {
	Enabled  bool
	MaxStack int
}

// StackingArray keeps track of active playback IDs for a specific track.
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
	trackID       TrackID
}
type TrackData struct {
	ext     string
	content []byte
}

// AudioManager manages all game audio, including music and sound effects.
type AudioManager struct {
	audioContext   *audio.Context
	trackList      map[TrackID]TrackData
	players        map[PlaybackID]*AudioSource
	stackingTracks map[TrackID]*StackingArray
	nextPlaybackID PlaybackID
}

// NewAudioManager initializes and returns a new AudioManager.
func NewAudioManager(sampleRate int) *AudioManager {
	return &AudioManager{
		audioContext:   audio.NewContext(sampleRate),
		trackList:      make(map[TrackID]TrackData),
		players:        make(map[PlaybackID]*AudioSource),
		nextPlaybackID: 0,
		stackingTracks: make(map[TrackID]*StackingArray),
	}
}

// fromBytes decodes audio from bytes.
func (self *AudioManager) fromBytes(content []byte, ext string) (io.ReadSeeker, error) {
	switch ext {
	case "ogg":
		return vorbis.DecodeF32(bytes.NewReader(content))
	case "wav":
		s, err := wav.DecodeF32(bytes.NewReader(content))
		if err != nil {
			// Provide a more helpful error message for the common PCM issue.
			return nil, fmt.Errorf("failed to decode .wav file. Ensure it is in Linear PCM format. You can convert it with ffmpeg: `ffmpeg -i input.wav -acodec pcm_s16le -ar 44100 output.wav`. Original error: %w", err)
		}
		return s, nil
	case "mp3":
		return mp3.DecodeF32(bytes.NewReader(content))
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", ext)
	}
}

// Load loads an audio file from disk and stores its bytes.
func (self *AudioManager) Load(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("audio file path cannot be empty")
	}
	b := readFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read audio file: %s", path)
	}
	id := TrackID(len(self.trackList))
	self.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// LoadEmbedded loads embedded audio data and stores its bytes.
func (self *AudioManager) LoadEmbedded(path string) (TrackID, error) {
	b := openEmbeddedFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read embedded audio file: %s", path)
	}
	id := TrackID(len(self.trackList))
	self.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// LoadFromAssetPacker loads audio from an asset pack and stores its bytes.
func (self *AudioManager) LoadFromAssetPacker(path string) (TrackID, error) {
	b := openBundledFile(path)
	if len(b) == 0 {
		return -1, fmt.Errorf("failed to read bundled audio file: %s", path)
	}
	id := TrackID(len(self.trackList))
	self.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// prepareAudioSource prepares a new audio source from stored bytes.
func (self *AudioManager) prepareAudioSource(trackID TrackID, pan float64, loop bool) (*AudioSource, error) {
	if int(trackID) < 0 || int(trackID) >= len(self.trackList) {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	trackData := self.trackList[trackID]
	reader, err := self.fromBytes(trackData.content, trackData.ext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode audio from bytes: %w", err)
	}
	var stream io.ReadSeeker = reader
	var panStream *StereoPanStream
	if loop {
		switch trackData.ext {
		case "ogg":
			vorbisStream, ok := reader.(*vorbis.Stream)
			if !ok {
				return nil, fmt.Errorf("failed to assert vorbis stream type")
			}
			panStream = NewStereoPanStream(audio.NewInfiniteLoop(vorbisStream, vorbisStream.Length()))
		case "wav":
			wavStream, ok := reader.(*wav.Stream)
			if !ok {
				return nil, fmt.Errorf("failed to assert wav stream type")
			}
			panStream = NewStereoPanStream(audio.NewInfiniteLoop(wavStream, wavStream.Length()))
		case "mp3":
			mp3Stream, ok := reader.(*mp3.Stream)
			if !ok {
				return nil, fmt.Errorf("failed to assert mp3 stream type")
			}
			panStream = NewStereoPanStream(audio.NewInfiniteLoop(mp3Stream, mp3Stream.Length()))
		default:
			return nil, fmt.Errorf("unsupported audio format for looping: %s", trackData.ext)
		}
	} else {
		panStream = NewStereoPanStream(stream)
	}
	if pan != 0 {
		panStream.SetPan(pan)
	}
	player, err := self.audioContext.NewPlayerF32(panStream)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio player: %w", err)
	}
	return &AudioSource{
		player:    player,
		panStream: panStream,
		trackID:   trackID,
	}, nil
}

// createAudioSource creates and initializes an audio source, starting playback.
func (self *AudioManager) createAudioSource(trackID TrackID, pan float64, loop bool, defaultVolume float64, fadeDuration float64, fadeType AudioFadeType) (*AudioSource, error) {
	source, err := self.prepareAudioSource(trackID, pan, loop)
	if err != nil {
		return nil, err
	}
	if fadeDuration <= 0 {
		source.currentVolume = defaultVolume
		source.player.SetVolume(defaultVolume)
		source.player.Play()
		return source, nil
	}
	startVolume := defaultVolume
	targetVolume := defaultVolume
	if fadeType == AudioFadeIn {
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
	return source, nil
}

// internalPlay handles playback logic, including stacking and fading.
func (self *AudioManager) internalPlay(trackID TrackID, pan float64, loop bool, defaultVolume float64, fadeDuration float64, fadeType AudioFadeType, stackConfig *StackingConfig) (PlaybackID, error) {
	stacking := stackConfig != nil && stackConfig.Enabled && stackConfig.MaxStack > 1
	if !stacking {
		self.StopByTrackID(trackID)
	} else {
		stackArray, ok := self.stackingTracks[trackID]
		if !ok {
			stackArray = &StackingArray{}
			self.stackingTracks[trackID] = stackArray
		}
		active := []PlaybackID{}
		for _, id := range stackArray.playbackIDs {
			if s, ok := self.players[id]; ok && s.player.IsPlaying() {
				active = append(active, id)
			}
		}
		stackArray.playbackIDs = active
		if len(active) >= stackConfig.MaxStack {
			oldest := active[0]
			if err := self.Stop(oldest); err != nil {
				log.Printf("error stopping oldest playback ID %d: %v\n", oldest, err)
			}
			stackArray.playbackIDs = active[1:]
		}
	}
	source, err := self.createAudioSource(trackID, pan, loop, defaultVolume, fadeDuration, fadeType)
	if err != nil {
		return -1, err
	}
	playbackID := self.nextPlaybackID
	self.players[playbackID] = source
	self.nextPlaybackID++
	if stacking {
		self.stackingTracks[trackID].playbackIDs = append(self.stackingTracks[trackID].playbackIDs, playbackID)
	}
	return playbackID, nil
}

// PlaySound plays a one-shot sound effect.
func (self *AudioManager) PlaySound(trackID TrackID, pan float64, stackConfig *StackingConfig) (PlaybackID, error) {
	return self.internalPlay(trackID, pan, false, 1.0, 0, AudioFadeIn, stackConfig)
}

// PlayMusic plays a music track.
func (self *AudioManager) PlayMusic(trackID TrackID, loop bool) (PlaybackID, error) {
	return self.internalPlay(trackID, 0, loop, 0.5, 0, AudioFadeIn, nil)
}

// FadeSound plays a sound effect with a fade effect.
func (self *AudioManager) FadeSound(trackID TrackID, pan, fadeDuration float64, fadeType AudioFadeType, stackConfig *StackingConfig) (PlaybackID, error) {
	return self.internalPlay(trackID, pan, false, 1.0, fadeDuration, fadeType, stackConfig)
}

// FadeMusic plays a music track with a fade effect.
func (self *AudioManager) FadeMusic(trackID TrackID, loop bool, fadeDuration float64, fadeType AudioFadeType) (PlaybackID, error) {
	return self.internalPlay(trackID, 0, loop, 1.0, fadeDuration, fadeType, nil)
}

// Stop stops and removes a single playing audio source.
func (self *AudioManager) Stop(id PlaybackID) error {
	source, ok := self.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	if stackArray, exists := self.stackingTracks[source.trackID]; exists {
		newPlaybackIDs := []PlaybackID{}
		for _, stackedID := range stackArray.playbackIDs {
			if stackedID != id {
				newPlaybackIDs = append(newPlaybackIDs, stackedID)
			}
		}
		stackArray.playbackIDs = newPlaybackIDs
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
	var idsToStop []PlaybackID
	for id, source := range self.players {
		if source.trackID == trackID {
			idsToStop = append(idsToStop, id)
		}
	}
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
			volumeChange := (1.0 / source.fadeDuration) * dt
			if source.fadeType == AudioFadeOut {
				volumeChange = -volumeChange
			}
			source.currentVolume += volumeChange
			fadeComplete := (source.fadeType == AudioFadeIn && source.currentVolume >= source.targetVolume) ||
				(source.fadeType == AudioFadeOut && source.currentVolume <= source.targetVolume)
			if fadeComplete {
				source.currentVolume = source.targetVolume
				source.isFading = false
			}
			source.player.SetVolume(source.currentVolume)
		}
		if !source.player.IsPlaying() && !source.isFading {
			if err := self.Stop(id); err != nil {
				log.Printf("error stopping playback ID %d: %v\n", id, err)
			}
		}
	}
}
