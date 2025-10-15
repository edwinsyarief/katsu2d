# Katsu2D (カツ2D)

Unleash your game ideas with this high-performance game framework built in Go, leveraging the power of Ebitengine. This framework features a fast and unique Entity Component System (ECS) architecture, an optimized batching renderer with support for position, rotation, and scaling, and a simplified API designed for both power and ease of use.

## Features

- **Lazy ECS Architecture**: Built on a high-performance, data-oriented, and cache-friendly [Entity Component System](https://github.com/edwinsyarief/teishoku).
- **Optimized Batch Renderer**: Efficiently draws multiple sprites in a single draw call, supporting position, rotation, and scaling.
- **Powerful Audio Manager**: Supports fade in/out, stereo panning, and sound stacking for complex audio scenes.
- **Flexible Line Renderer**: Easily draw lines with color and width interpolation, join, and cap options.
- **Flexible Ribbon Trail**: Create smooth, ribbon-like trails for dynamic visual effects.
- **Modular and Extensible**: Designed for easy extension, allowing you to add new components and systems.
- **Rich Set of Built-in Components**: Includes components for sprites, text, animation, input, and more, ready to use out-of-the-box.
- **And much more!**

## Getting Started

The engine is designed to be highly configurable and easy to use through its engine.NewEngine constructor and functional options.

### Prerequisites

- Go 1.21 or higher
- [Ebitengine](https://ebitengine.org) v2.8.8 (or compatible version)

### Installation

```cmd
go get github.com/edwinsyarief/katsu2d
```

### Usage

For detailed examples and usage, please see the [katsu2d-examples](https://github.com/edwinsyarief/katsu2d-examples) repository.

### Functional options

The `katsu2d.NewEngine` function accepts various Option functions to configure the engine:

- `katsu2d.WithWindowSize(int, int)`: Overrides the default window dimensions.
- `katsu2d.WithWindowTitle(string)`: Overrides the default window title.
- `katsu2d.WithWindowsResizeMode(ebiten.WindowResizingModeType)`: Overrides the default windows resizing mode.
- `katsu2d.WithFullScreen(bool)`: Overrides the default windows fullscreen mode.
- `katsu2d.WithVsyncEnabled(bool)`: Overrides the default vsync mode.
- `katsu2d.WithCursorMode(ebiten.CursorModeType)`: Overrides the default cursor mode.
- `katsu2d.WithClearScreenEachFrame(bool)`: Overrides the default clear screen every frame.

### Managing Assets (Textures and Fonts)

It's a common and recommended practice to load all your necessary assets, like images and fonts, when the game first starts. This prevents stuttering and lag during gameplay that can be caused by loading files from disk.

Katsu2D provides a `TextureManager` and a `FontManager` accessible from the `Engine` object. You can use these to load your assets and get back a numeric ID for each one. You can then store these IDs in a global `assets` package or struct for easy access throughout your game.

Here is an example of a function that loads several assets at initialization:

```go
// A struct to hold all our asset IDs
var (
    EbitengineLogoTextureID int
    DefaultFontID           int
    // ... add more asset IDs here
)

func loadAssets(e *katsu2d.Engine) {
    // Load textures and get their IDs
    EbitengineLogoTextureID = e.TextureManager().LoadEmbedded("embedded/images/ebitengine_logo.png")
    
    // Load fonts and get their IDs
    DefaultFontID = e.FontManager().LoadEmbedded("embedded/fonts/default.ttf")
}
```

By calling a function like this at startup, you can refer to your assets using the stored IDs (e.g., `assets.EbitengineLogoTextureID`) when creating components later.

## Why Katsu2D?

The name **Katsu2D** draws a playful and fitting analogy to the beloved Japanese dish (カツ), often a perfectly breaded and fried cutlet.

Just as a great katsu is known for its:

- **Crispy, High-Performance Exterior:** The framework aims for **blazing fast performance** and a **sharp, responsive feel** in your 2D games, much like the satisfying crunch of a well-fried panko crust.
- **Tender, Robust Interior:** Beneath the surface, Katsu2D offers a **robust and reliable ECS core**, providing a solid foundation for complex game logic and interactions, akin to the tender, flavorful meat within.
- **Simple, Satisfying Experience:** The goal is to provide a **streamlined and enjoyable development experience**, allowing you to focus on the creative aspects of your game without getting bogged down in boilerplate, just as a perfectly prepared katsu offers a simple yet deeply satisfying meal.

Combined with "2D" to clearly define its domain, **Katsu2D** signifies a framework designed to help you create games that are both **performant and a pleasure to build and play**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
