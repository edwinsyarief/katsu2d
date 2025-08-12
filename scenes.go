package katsu2d

import (
	"fmt"
	"image/color"
	"os"
	"sync"
	"time"

	"github.com/edwinsyarief/katsu2d/logger"
	"github.com/edwinsyarief/katsu2d/overlays"
	"github.com/hajimehoshi/ebiten/v2"
)

// Scene defines the interface for game scenes.
type Scene interface {
	OnEnter(*Engine)    // Called when entering the scene (load/setup)
	Update(float64)     // Update logic (delegates to systems)
	Draw(*ebiten.Image) // Draw logic (delegates to systems)
	OnExit()            // Called when exiting the scene (cleanup)
}

// SceneManager manages scene switching.
type SceneManager struct {
	current Scene // Current active scene
	next    Scene // Next scene to switch to
}

// Update handles scene switching and updates the current scene.
func (self *SceneManager) Update(dt float64, engine *Engine) {
	if self.next != nil {
		if self.current != nil {
			self.current.OnExit()
		}
		self.current = self.next
		self.next = nil
		self.current.OnEnter(engine)
	}
	if self.current != nil {
		self.current.Update(dt)
	}
}

// Draw draws the current scene.
func (self *SceneManager) Draw(screen *ebiten.Image) {
	if self.current != nil {
		self.current.Draw(screen)
	}
}

// SetScene queues a scene switch for the next update.
func (self *SceneManager) SetScene(s Scene) {
	self.next = s
}

func (self *SceneManager) SetLoadingScene(task func() []any, callback func([]any), drawFunc func(*ebiten.Image)) {
	loading := &LoadingScene{
		task:         task,
		callback:     callback,
		drawFunc:     drawFunc,
		fadeInSpeed:  1.75,
		fadeOutSpeed: 1.15,
	}

	self.SetScene(loading)
}

type BaseScene struct {
	World   *World
	Systems []System
}

func (self *BaseScene) Update(dt float64) {
	for _, sys := range self.Systems {
		sys.Update(self.World, dt)
	}
}

func (self *BaseScene) Draw(screen *ebiten.Image) {
	for _, sys := range self.Systems {
		sys.Draw(self.World, screen)
	}
}

func (self *BaseScene) OnExit() {
	self.Systems = nil
	self.World = nil
}

// maxRetries defines the maximum number of retry attempts for the task.
const maxRetries = 3

// LoadingScene displays a loading screen while executing a task asynchronously.
// It transitions to the next scene upon task completion, with a fade effect.
type LoadingScene struct {
	BaseScene

	args                      []any        // Arguments to pass to the next scene
	loading                   bool         // Indicates if the task is running
	task                      func() []any // Task to execute during loading
	callback                  func([]any)
	retryCount                int // Number of retry attempts
	fadeColor                 color.RGBA
	startTask                 bool                // Flag to start the task after fade-in
	drawFunc                  func(*ebiten.Image) // provide this to display something in the loading scene
	mutex                     sync.Mutex          // Protects loading state
	fadeInSpeed, fadeOutSpeed float64             // fade speed
	engine                    *Engine
}

// Init initializes the loading scene, setting up the text label and fade overlay.
func (self *LoadingScene) OnEnter(engine *Engine) {
	self.engine = engine
	self.World = NewWorld()
	self.Systems = []System{
		NewFadeOverlaySystem(),
	}

	e := self.World.NewEntity()
	self.World.AddComponent(e, overlays.NewFadeOverlay(
		self.engine.ScreenWidth(),
		self.engine.ScreenHeight(),
		overlays.FadeTypeIn,
		self.fadeColor,
		self.fadeInSpeed,
		self.onFadeIn,
	))
}

// Update advances the scene’s state, managing the fade overlay and task execution.
func (self *LoadingScene) Update(dt float64) {
	self.BaseScene.Update(dt)

	if self.task == nil || !self.startTask {
		return
	}
	self.mutex.Lock()
	if self.loading {
		self.mutex.Unlock()
		return
	}
	self.loading = true
	self.mutex.Unlock()
	// Execute task asynchronously
	go self.executeTask()
}

// Draw renders the loading text and fade overlay to the screen.
func (self *LoadingScene) Draw(screen *ebiten.Image) {
	if self.drawFunc != nil {
		self.drawFunc(screen)
	}
	self.BaseScene.Draw(screen)
}

// executeTask runs the loading task, handling retries and errors.
func (self *LoadingScene) executeTask() {
	defer func() {
		self.mutex.Lock()
		self.loading = false
		self.mutex.Unlock()
	}()
	for {
		result, err := self.runTask()
		if err == nil {
			self.mutex.Lock()
			self.args = result
			self.task = nil
			self.mutex.Unlock()
			e := self.World.QueryAll(CTFadeOverlay)[0]
			fade := self.World.GetComponent(e, CTFadeOverlay).(*overlays.FadeOverlay)
			fade.Reset(overlays.FadeTypeOut, color.RGBA{R: 0, G: 0, B: 0, A: 255}, 1, self.fadeOutSpeed, self.onFadeOut)
			return
		}
		self.retryCount++
		logger.GetLogger().Error("Loading task failed: %v, attempt %d/%d", err, self.retryCount, maxRetries)
		if self.retryCount >= maxRetries {
			logger.GetLogger().Error("Max retries reached for loading task. Aborting.")
			os.Exit(1)
		}
		time.Sleep(2 * time.Second)
	}
}

// runTask executes the task and captures any panics or errors.
func (self *LoadingScene) runTask() (result []any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in loading task: %v", r)
			logger.GetLogger().Error("Panic in loading task: %v", err)
		}
	}()
	result = self.task()
	return result, nil
}

// onFadeIn is called when the fade-in effect completes, starting the task.
func (self *LoadingScene) onFadeIn() {
	self.startTask = true
}

// onFadeOut is called when the fade-out effect completes, transitioning to the next scene.
func (self *LoadingScene) onFadeOut() {
	if self.callback != nil {
		self.callback(self.args)
	}
}
