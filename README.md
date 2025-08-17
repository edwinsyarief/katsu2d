# Katsu2D (カツ2D)

Unleash your game ideas with this high-performance game framework built in Go, leveraging the power of Ebitengine. This framework features a fast and unique Entity Component System (ECS) architecture, an optimized batching renderer with support for position, rotation, and scaling, and a simplified API designed for both power and ease of use.

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
- **Built-in components:** Too lazy to create sprite, text, or other components? No worries, we have built-int components and systems that are ready to use.
- **3-Layer Tilemap System:** A high-performance, JSON-driven tilemap system with three distinct layers: a background layer, a Y-sorted middle layer for dynamic objects, and a foreground overlay layer.

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

Here is the example repo: [https://github.com/edwinsyarief/katsu2d-simple-demo](https://github.com/edwinsyarief/katsu2d-simple-demo)

#### Screenshot

![Simple Demo](./screenshot.png)

### Functional options

The `katsu2d.NewEngine` function accepts various Option functions to configure the engine:

- `katsu2d.WithWindowSize(int, int)`: Overrides the default window dimensions.
- `katsu2d.WithWindowTitle(string)`: Overrides the default window title.
- `katsu2d.WithWindowsResizeMode(ebiten.WindowResizingModeType)`: Overrides the default windows resizing mode.
- `katsu2d.WithFullScreen(bool)`: Overrides the default windows fullscreen mode.
- `katsu2d.WithVsyncEnabled(bool)`: Overrides the default vsync mode.
- `katsu2d.WithCursorMode(ebiten.CursorModeType)`: Overrides the default cursor mode.
- `katsu2d.WithClearScreenEachFrame(bool)`: Overrides the default clear screen every frame.

### Working with Entities and Components

TODO

### 3-Layer Tilemap System

The engine includes a powerful and easy-to-use 3-Layer Tilemap system. It allows you to create complex maps with a background, a dynamic middle ground, and a foreground overlay, all loaded from a simple JSON format.

#### Creating a Tilemap from JSON

The most common way to create a tilemap is by loading it from a JSON file.

```go
import (
    "os"
    "github.com/edwinsyarief/katsu2d/dualgrid"
)

// Read the map file
data, err := os.ReadFile("path/to/your/map.json")
if err != nil {
    // handle error
}

// Load the tilemap
tilemap, err := dualgrid.LoadFromJSON(data)
if err != nil {
    // handle error
}

// You can now access the grids
grassTile := tilemap.LowerGrid.Get(0, 0)
treeTile := tilemap.UpperGrid.Get(1, 1)
```

#### Accessing Tiles

You can get a tile from either grid using the `Get(x, y)` method.

```go
// Get the tile at coordinates (2, 3) from the lower grid
tile := tilemap.LowerGrid.Get(2, 3)

if tile != nil {
    // Access tile properties
    walkable := tile.Properties["walkable"].(bool)
    name := tile.Properties["name"].(string)
}
```

#### JSON Map Format

The JSON file has a specific structure that the `LoadFromJSON` function expects.

*   `width`, `height`: The dimensions of the map in tiles.
*   `tileset`: An object where keys are tile IDs (as strings) and values are objects containing the tile's properties.
*   `lower_grid`, `upper_grid`, `top_grid`: Arrays of tile IDs representing the grid layout. The length of each array must be `width * height`. Use `0` for an empty tile. The `top_grid` is optional for backward compatibility.

##### Example JSON (`map.json`)

```json
{
  "width": 20,
  "height": 15,
  "tileset": {
    "1": { "name": "grass" },
    "2": { "name": "water" },
    "3": { "name": "tree_trunk" },
    "4": { "name": "tree_leaves" },
    "5": { "name": "path" }
  },
  "lower_grid": [ 1, 1, 5, 5, "..." ],
  "upper_grid": [ 0, 3, 0, 0, "..." ],
  "top_grid":   [ 0, 4, 0, 0, "..." ]
}
```

#### ECS Integration

The 3-layer rendering pipeline uses three separate systems working together. The order in which they are added to the engine is important.

1.  **`TileMapRenderSystem` (Background)**: This system draws the `LowerGrid` first, acting as the static background. It should be added as a `BackgroundDrawSystem`.
2.  **`YSortedRenderSystem` (Background)**: This system collects all sprites and all tiles from the `UpperGrid`, sorts them together by their Z-layer and Y-position, and then draws them. This creates the illusion of depth. It should also be added as a `BackgroundDrawSystem` and run *after* the `TileMapRenderSystem`.
3.  **`TopGridRenderSystem` (Overlay)**: This system draws the `TopGrid` last. It should be added as an `OverlayDrawSystem` to ensure it always appears on top of all other game elements.

**Example System Setup:**

```go
// Get the texture manager from the engine
tm := engine.TextureManager()

// Add the rendering systems in the correct order
engine.AddBackgroundDrawSystem(katsu2d.NewTileMapRenderSystem(tm))
engine.AddBackgroundDrawSystem(katsu2d.NewYSortedRenderSystem(tm))
engine.AddOverlayDrawSystem(katsu2d.NewTopGridRenderSystem(tm))
```

## Why Katsu2D?

The name **Katsu2D** draws a playful and fitting analogy to the beloved Japanese dish (カツ), often a perfectly breaded and fried cutlet.

Just as a great katsu is known for its:

- **Crispy, High-Performance Exterior:** The framework aims for **blazing fast performance** and a **sharp, responsive feel** in your 2D games, much like the satisfying crunch of a well-fried panko crust.
- **Tender, Robust Interior:** Beneath the surface, Katsu2D offers a **robust and reliable ECS core**, providing a solid foundation for complex game logic and interactions, akin to the tender, flavorful meat within.
- **Simple, Satisfying Experience:** The goal is to provide a **streamlined and enjoyable development experience**, allowing you to focus on the creative aspects of your game without getting bogged down in boilerplate, just as a perfectly prepared katsu offers a simple yet deeply satisfying meal.

Combined with "2D" to clearly define its domain, **Katsu2D** signifies a framework designed to help you create games that are both **performant and a pleasure to build and play**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
