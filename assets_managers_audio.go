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
func (s *StereoPanStream) Read(p []byte) (int, error) {
	// If the stream has a buffer that was read in the previous time, use this first.
	var bufN int
	if len(s.buf) > 0 {
		bufN = copy(p, s.buf)
		rem := len(s.buf) - bufN
		if rem > 0 {
			copy(s.buf, s.buf[bufN:])
		}
		s.buf = s.buf[:rem]
	}
	readN, err := s.ReadSeeker.Read(p[bufN:])
	if err != nil && err != io.EOF {
		return 0, err
	}
	// Align the buffer size in multiples of 8 (4 bytes per channel, 2 channels).
	totalN := bufN + readN
	extra := totalN % 8
	s.buf = append(s.buf, p[totalN-extra:totalN]...)
	alignedN := totalN - extra
	// Calculate stereo balance using Unity's approach
	ls := float32(math.Min(s.pan*-1+1, 1))
	rs := float32(math.Min(s.pan+1, 1))
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
func (s *StereoPanStream) SetPan(pan float64) {
	s.pan = math.Min(math.Max(-1, pan), 1)
}

// Pan returns the current pan value of the stream.
func (s *StereoPanStream) Pan() float64 {
	return s.pan
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

// TrackData holds the decoded format extension and raw audio bytes for a loaded track.
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
	finishedIDs    []PlaybackID
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
func (am *AudioManager) fromBytes(content []byte, ext string) (io.ReadSeeker, error) {
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
func (am *AudioManager) Load(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("audio file path cannot be empty")
	}
	b, err := readFileErr(path)
	if err != nil {
		return -1, fmt.Errorf("failed to read audio file %s: %w", path, err)
	}
	if len(b) == 0 {
		return -1, fmt.Errorf("audio file is empty: %s", path)
	}
	id := TrackID(len(am.trackList))
	am.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// LoadEmbedded loads embedded audio data and stores its bytes.
func (am *AudioManager) LoadEmbedded(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("embedded audio file path cannot be empty")
	}
	b, err := openEmbeddedFileErr(path)
	if err != nil {
		return -1, fmt.Errorf("failed to read embedded audio file %s: %w", path, err)
	}
	if len(b) == 0 {
		return -1, fmt.Errorf("embedded audio file is empty: %s", path)
	}
	id := TrackID(len(am.trackList))
	am.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// LoadFromAssetPacker loads audio from an asset pack and stores its bytes.
func (am *AudioManager) LoadFromAssetPacker(path string) (TrackID, error) {
	if path == "" {
		return -1, fmt.Errorf("bundled audio file path cannot be empty")
	}
	b, err := openBundledFileErr(path)
	if err != nil {
		return -1, fmt.Errorf("failed to read bundled audio file %s: %w", path, err)
	}
	if len(b) == 0 {
		return -1, fmt.Errorf("bundled audio file is empty: %s", path)
	}
	id := TrackID(len(am.trackList))
	am.trackList[id] = TrackData{content: b, ext: GetFileExtension(path)}
	return id, nil
}

// prepareAudioSource prepares a new audio source from stored bytes.
func (am *AudioManager) prepareAudioSource(trackID TrackID, pan float64, loop bool) (*AudioSource, error) {
	trackData, ok := am.trackList[trackID]
	if !ok {
		return nil, fmt.Errorf("invalid track ID: %d", trackID)
	}
	reader, err := am.fromBytes(trackData.content, trackData.ext)
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
	player, err := am.audioContext.NewPlayerF32(panStream)
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
func (am *AudioManager) createAudioSource(trackID TrackID, pan float64, loop bool, defaultVolume float64, fadeDuration float64, fadeType AudioFadeType) (*AudioSource, error) {
	source, err := am.prepareAudioSource(trackID, pan, loop)
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
func (am *AudioManager) internalPlay(trackID TrackID, pan float64, loop bool, defaultVolume float64, fadeDuration float64, fadeType AudioFadeType, stackConfig *StackingConfig) (PlaybackID, error) {
	stacking := stackConfig != nil && stackConfig.Enabled && stackConfig.MaxStack > 1
	if !stacking {
		am.StopByTrackID(trackID)
	} else {
		stackArray, ok := am.stackingTracks[trackID]
		if !ok {
			stackArray = &StackingArray{}
			am.stackingTracks[trackID] = stackArray
		}
		active := stackArray.playbackIDs[:0]
		for _, id := range stackArray.playbackIDs {
			if s, ok := am.players[id]; ok && s.player.IsPlaying() {
				active = append(active, id)
			}
		}
		stackArray.playbackIDs = active
		if len(active) >= stackConfig.MaxStack {
			oldest := active[0]
			if err := am.Stop(oldest); err != nil {
				log.Printf("error stopping oldest playback ID %d: %v\n", oldest, err)
			}
		}
	}
	source, err := am.createAudioSource(trackID, pan, loop, defaultVolume, fadeDuration, fadeType)
	if err != nil {
		return -1, err
	}
	playbackID := am.nextPlaybackID
	am.players[playbackID] = source
	am.nextPlaybackID++
	if stacking {
		sa, ok := am.stackingTracks[trackID]
		if !ok {
			sa = &StackingArray{}
			am.stackingTracks[trackID] = sa
		}
		sa.playbackIDs = append(sa.playbackIDs, playbackID)
	}
	return playbackID, nil
}

// PlaySound plays a one-shot sound effect.
func (am *AudioManager) PlaySound(trackID TrackID, pan float64, stackConfig *StackingConfig) (PlaybackID, error) {
	return am.internalPlay(trackID, pan, false, 1.0, 0, AudioFadeIn, stackConfig)
}

// PlayMusic plays a music track.
func (am *AudioManager) PlayMusic(trackID TrackID, loop bool) (PlaybackID, error) {
	return am.internalPlay(trackID, 0, loop, 0.5, 0, AudioFadeIn, nil)
}

// FadeSound plays a sound effect with a fade effect.
func (am *AudioManager) FadeSound(trackID TrackID, pan, fadeDuration float64, fadeType AudioFadeType, stackConfig *StackingConfig) (PlaybackID, error) {
	return am.internalPlay(trackID, pan, false, 1.0, fadeDuration, fadeType, stackConfig)
}

// FadeMusic plays a music track with a fade effect.
func (am *AudioManager) FadeMusic(trackID TrackID, loop bool, fadeDuration float64, fadeType AudioFadeType) (PlaybackID, error) {
	return am.internalPlay(trackID, 0, loop, 1.0, fadeDuration, fadeType, nil)
}

// Stop stops and removes a single playing audio source.
func (am *AudioManager) Stop(id PlaybackID) error {
	source, ok := am.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	if stackArray, exists := am.stackingTracks[source.trackID]; exists {
		n := 0
		for _, stackedID := range stackArray.playbackIDs {
			if stackedID != id {
				stackArray.playbackIDs[n] = stackedID
				n++
			}
		}
		stackArray.playbackIDs = stackArray.playbackIDs[:n]
		if len(stackArray.playbackIDs) == 0 {
			delete(am.stackingTracks, source.trackID)
		}
	}
	source.player.Pause()
	source.player.Close()
	delete(am.players, id)
	return nil
}

// StopByTrackID stops all playing instances of a specific track.
func (am *AudioManager) StopByTrackID(trackID TrackID) {
	am.finishedIDs = am.finishedIDs[:0]
	for id, source := range am.players {
		if source.trackID == trackID {
			am.finishedIDs = append(am.finishedIDs, id)
		}
	}
	for _, id := range am.finishedIDs {
		_ = am.Stop(id)
	}
}

// Pause pauses a single playing audio source, allowing it to be resumed later.
func (am *AudioManager) Pause(id PlaybackID) error {
	source, ok := am.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.player.Pause()
	return nil
}

// Resume resumes a paused audio source.
func (am *AudioManager) Resume(id PlaybackID) error {
	source, ok := am.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.player.Play()
	return nil
}

// IsPlaying checks if a specific playback instance is currently playing.
func (am *AudioManager) IsPlaying(id PlaybackID) bool {
	source, ok := am.players[id]
	if !ok {
		return false
	}
	return source.player.IsPlaying()
}

// IsTrackPlaying checks if any instance of a given track ID is currently playing.
func (am *AudioManager) IsTrackPlaying(trackID TrackID) bool {
	for _, source := range am.players {
		if source.trackID == trackID && source.player.IsPlaying() {
			return true
		}
	}
	return false
}

// StopAll stops all currently playing audio sources.
func (am *AudioManager) StopAll() {
	am.finishedIDs = am.finishedIDs[:0]
	for id := range am.players {
		am.finishedIDs = append(am.finishedIDs, id)
	}
	for _, id := range am.finishedIDs {
		_ = am.Stop(id)
	}
}

// SetVolume adjusts the volume of a single playing audio source.
func (am *AudioManager) SetVolume(id PlaybackID, volume float64) error {
	source, ok := am.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.currentVolume = volume
	source.player.SetVolume(volume)
	return nil
}

// SetPan adjusts the stereo pan of a single playing audio source.
func (am *AudioManager) SetPan(id PlaybackID, pan float64) error {
	source, ok := am.players[id]
	if !ok {
		return fmt.Errorf("invalid playback ID: %d", id)
	}
	source.panStream.SetPan(pan)
	return nil
}

// Update handles audio state updates and cleanup.
func (am *AudioManager) Update(dt float64) {
	am.finishedIDs = am.finishedIDs[:0]
	for id, source := range am.players {
		if source.isFading {
			if source.fadeDuration <= 0 {
				source.currentVolume = source.targetVolume
				source.isFading = false
				source.player.SetVolume(source.currentVolume)
			} else {
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
		}
		if !source.player.IsPlaying() && !source.isFading {
			am.finishedIDs = append(am.finishedIDs, id)
		}
	}
	for _, id := range am.finishedIDs {
		if err := am.Stop(id); err != nil {
			log.Printf("error stopping playback ID %d: %v\n", id, err)
		}
	}
}
