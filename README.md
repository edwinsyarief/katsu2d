# Katsu2D

Unleash your game ideas with this high-performance game engine built in Go, leveraging the power of Ebitengine. This engine features a fast and unique Entity Component System (ECS) architecture, an optimized batching renderer with support for position, rotation, and scaling, and a simplified API designed for both power and ease of use.

## 🚧 WORK IN PROGRESS 🚧

This project is currently under active development. While core functionalities are in place and demonstrate high performance, the API and internal structures may undergo significant changes. It is not yet recommended for production use.

**Your feedback and contributions are highly welcome!** If you encounter issues, have suggestions, or wish to contribute, please feel free to do so.

## Features

- **Fast & Unique ECS Architecture:** A robust Entity Component System that promotes modularity, reusability, and high performance by separating data (components) from logic (systems).
- **Optimized Batching Renderer:** Efficiently draws multiple sprites in a single draw call, significantly reducing overhead. Supports advanced transformations like rotation and scaling for batched entities.
- **Smooth Slow Motion Control:** Implement dynamic time scaling with smooth interpolation (lerping) between normal and slow-motion speeds, allowing for compelling gameplay effects.
- **Flexible Cooldown System:** Easily manage ability cooldowns or timed events with a dedicated component and system.
- **Simplified API with Functional Options:** A fluent, chainable API for engine configuration and entity creation, inspired by the "Functional Options" pattern, making setup and game logic development more intuitive.
- **Modular and Extensible:** Designed with clear separation of concerns, making it easy to add new components, systems, or features.

## Getting Started

The engine is designed to be highly configurable and easy to use through its engine.NewEngine constructor and functional options.

### Prerequisites

- Go 1.21 or higher
- [Ebitengine](https://ebitengine.org) v2.8.8 (or compatible version)

### Installation

```cmd
go get github.com/edwinsyarief/katsu2d
```

### Usage example

```go

import (
	"katsu2d"
	"katsu2d/components"
	"katsu2d/constants"
	"katsu2d/scene"
	"katsu2d/systems"
	
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	game := katsu2d.NewEngine(
		katsu2d.WithWindowSize(800, 600),
		katsu2d.WithWindowTitle("Katsu2D Demo"),
		katsu2d.WithWindowsResizeMode(ebiten.WindowResizingModeEnabled),
		katsu2d.WithWindowFullscreen(false),
		katsu2d.WithWindowVsync(true),
		katsu2d.WithCursorMode(ebiten.CursorModeHidden),
		katsu2d.WithScreenClearedEveryFrame(false),
	)

	game.SetTimeScale(1) // Set the time scale to 100% for normal speed

	// Create default scene
	defaultScene := scene.NewDefaultScene()
	game.SceneManager().AddScene("default", defaultScene)
	game.SceneManager().SwitchScene("default")

	// Register systems
	game.World().AddSystem(systems.NewUpdateSystem())
	game.World().AddSystem(systems.NewBatchSystem())
	game.World().AddSystem(systems.NewRenderSystem())

	// Create a batched sprite entity (red square)
	// This entity will now be rendered with the new batcher, but with no rotation.
	entity1 := game.World().CreateEntity()
	t := katsu2d.T()
	t.SetPosition(katsu2d.V2(100))
	world.AddComponent(entity1, t)
	world.AddComponent(entity1, &components.DrawableBatch{
		TextureID: 0, // Assuming texture ID 0 is a placeholder
		Width:     64,
		Height:    64,
		Color:     color.RGBA{R: 255, G: 0, B: 0, A: 255}, // Red
	})

	if err := game.RunGame(); err != nil {
		panic(err)
	}
}
```

### Functional options

The `katsu2d.NewEngine` function accepts various Option functions to configure the engine:

- `katsu2d.WithWindowSize(int, int)`: Overrides the default window dimensions.
- `katsu2d.WithWindowTitle(string)`: Overrides the default window title.
- `katsu2d.WithWindowsResizeMode(ebiten.WindowResizingModeType)`: Overrides the default windows resizing mode.
- `katsu2d.WithWindowFullscreen(bool)`: Overrides the default windows fullscreen mode.
- `katsu2d.WithWindowVsync(bool)`: Overrides the default vsync mode.
- `katsu2d.WithCursorMode(ebiten.CursorModeType)`: Overrides the default cursor mode.
- `katsu2d.WithScreenClearedEveryFrame(bool)`: Overrides the default clear screen every frame.

### Working with Entities and Components

TODO

#### Important Note for Updatable Components

The `UpdateFunc` in the `components.Updatable` receives `dt float64`, `world *ecs.World`, and `entityID ecs.EntityID` as parameters. This allows your entity's logic to interact directly with the ECS world and its own components.

### Slow Motion and Cooldowns

TODO

## Why Katsu2D?

The name Katsu2D is chosen to reflect the core philosophy and aspirations of this game engine.

- **"Katsu" (勝つ):** In Japanese, "Katsu" means "to win" or "to be victorious." This conveys the engine's goal of helping developers succeed in creating high-performance 2D games. It also evokes a sense of speed and efficiency, which are central to the engine's design, particularly its optimized batching renderer and ECS architecture.
- **"2D":** This clearly indicates the engine's focus on two-dimensional game development, making its purpose immediately clear to potential users.

Together, **Katsu2D** aims to be a concise and memorable name that signifies a **victorious and efficient platform for 2D game creation**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
