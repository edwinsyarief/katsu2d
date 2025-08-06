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

	// Create an updatable entity with a batched drawable (blue square)
	// This entity will now visibly rotate and scale, demonstrating the new batcher's capabilities.
	entity2 := game.World().CreateEntity()
	t2 := katsu2d.TPos(300, 300)
	world.AddComponent(entity2, t2)
	world.AddComponent(entity2, &components.Updatable{
		UpdateFunc: func(dt float64, w *ecs.World, eID ecs.EntityID) {
			// Simple rotation effect
			if component, exists := w.GetComponent(eID, constants.ComponentTransform); exists {
				if transform, ok := component.(*components.Transform); ok {
					transform.Rotate(dt)
					if transform.Rotation() > 2*math.Pi {
						transform.Rotate(-2 * math.Pi)
					}
					// Simple scaling effect
					transform.SetScale(
						katsu2d.V(
							1.0+float64(math.Sin(float64(transform.Rotation())*2.0))*0.5,
							1.0+float64(math.Sin(float64(transform.Rotation())*2.0))*0.5,
						),
					)
				}
			}
		},
	})
	world.AddComponent(entity2, &components.DrawableBatch{
		TextureID: 0,
		Width:     32,
		Height:    32,
		Color:     color.RGBA{R: 0, G: 0, B: 255, A: 255}, // Blue
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

The name **Katsu2D** draws a playful and fitting analogy to the beloved Japanese dish, often a perfectly breaded and fried cutlet.

Just as a great katsu is known for its:

- **Crispy, High-Performance Exterior:** The engine aims for **blazing fast performance** and a **sharp, responsive feel** in your 2D games, much like the satisfying crunch of a well-fried panko crust.
- **Tender, Robust Interior:** Beneath the surface, Katsu2D offers a **robust and reliable ECS core**, providing a solid foundation for complex game logic and interactions, akin to the tender, flavorful meat within.
- **Simple, Satisfying Experience:** The goal is to provide a **streamlined and enjoyable development experience**, allowing you to focus on the creative aspects of your game without getting bogged down in boilerplate, just as a perfectly prepared katsu offers a simple yet deeply satisfying meal.

Combined with "2D" to clearly define its domain, **Katsu2D** signifies an engine designed to help you create games that are both **performant and a pleasure to build and play**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
