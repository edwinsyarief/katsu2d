# Katsu2D (カツ2D)

Unleash your game ideas with this high-performance game framework built in Go, leveraging the power of Ebitengine. This framework features a fast and unique Entity Component System (ECS) architecture, an optimized batching renderer with support for position, rotation, and scaling, and a simplified API designed for both power and ease of use.

## 🚧 WORK IN PROGRESS 🚧

This project is currently under active development. While core functionalities are in place and demonstrate high performance, the API and internal structures may undergo significant changes. It is not yet recommended for production use.

**Your feedback and contributions are highly welcome!** If you encounter issues, have suggestions, or wish to contribute, please feel free to do so.

## Features

- **Fast & Unique ECS Architecture:** A robust Entity Component System that promotes modularity, reusability, and high performance by separating data (components) from logic (systems).
- **Optimized Batching Renderer:** Efficiently draws multiple sprites in a single draw call, significantly reducing overhead. Supports advanced transformations like rotation and scaling for batched entities.
- **Powerful Audio Manager:** Feature-rich audio system supporting music and sound effects with:
  - Fade in/out transitions for smooth audio blending
  - Stereo panning for positional audio
  - Sound stacking for multiple concurrent instances
  - Independent volume control for each audio source
- **Smooth Slow Motion Control:** Implement dynamic time scaling with smooth interpolation (lerping) between normal and slow-motion speeds, allowing for compelling gameplay effects.
- **Flexible Line Renderer:** 2D polyline, easily draw line with color and width interpolation, join and cap options, etc.
- **Flexible Cooldown System:** Easily manage ability cooldowns or timed events with a dedicated component and system.
- **Simplified API with Functional Options:** A fluent, chainable API for engine configuration and entity creation, inspired by the "Functional Options" pattern, making setup and game logic development more intuitive.
- **Modular and Extensible:** Designed with clear separation of concerns, making it easy to add new components, systems, or features.
- **Built-in components:** Too lazy to create sprite, text, or other components? No worries, we have built-int components and systems that are ready to use.

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

Various examples repo: [https://github.com/edwinsyarief/katsu2d-examples](https://github.com/edwinsyarief/katsu2d-examples)

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

### Working with Entities and Components

The ECS (Entity Component System) is the core of Katsu2D. It allows you to organize your game objects in a data-oriented way, which is great for performance and keeping your code clean.

#### Creating an Entity

An entity is just a unique ID. You can create one using the `world.CreateEntity()` method:

```go
// Assuming 'world' is your *katsu2d.World instance
entity := world.CreateEntity()
```

#### Adding a Component

Once you have an entity, you can add components to it using constructor functions.

**TransformComponent:**
The `TransformComponent` manages an entity's position, scale, and rotation.

```go
// Create a new transform component
t := katsu2d.NewTransformComponent()
// Set its position in the world
t.SetPosition(ebimath.V(consts.WindowWidth/2, consts.WindowHeight/2))
// Add the component to the entity
world.AddComponent(entity, t)
```

**SpriteComponent:**
The `SpriteComponent` allows an entity to be rendered as a sprite. It requires a `textureID` that you get from the `TextureManager` when you load your assets.

```go
// Get the texture info from the manager
// (Assuming EbitengineLogoTextureID was loaded as shown in the "Managing Assets" section)
textureInfo := e.TextureManager().Get(EbitengineLogoTextureID)

// Create the sprite component using the texture ID and the texture's bounds
s := katsu2d.NewSpriteComponent(EbitengineLogoTextureID, textureInfo.Bounds())

// Add the component to the entity
world.AddComponent(entity, s)
```

#### Component IDs

For operations like removing components or querying entities, you need a way to refer to a component's *type*. Katsu2D provides a set of built-in constants for this purpose. Each constant represents a unique `ComponentID`.

Here are some of the most common ones:

- `katsu2d.CTTransform`: ID for `TransformComponent`
- `katsu2d.CTSprite`: ID for `SpriteComponent`
- `katsu2d.CTAnimation`: ID for `AnimationComponent`
- `katsu2d.CTText`: ID for `TextComponent`
- `katsu2d.CTInput`: ID for `InputComponent`

You can find the full list of built-in component IDs in `components.go`.

#### Removing a Component

You can remove a component from an entity using its `ComponentID`.

```go
// Remove the TransformComponent from the entity
world.RemoveComponent(entity, katsu2d.CTTransform)
```

#### Filtering Entities (Querying)

Systems use queries to find entities that have a specific set of components. The `world.Query()` method takes one or more `ComponentID`s and returns a slice of all entities that have *all* of the specified components.

```go
// Find all entities that have both a TransformComponent and a SpriteComponent
entities := world.Query(katsu2d.CTTransform, katsu2d.CTSprite)

// You can now loop through the returned entities
for _, entity := range entities {
    // Get a specific component from the entity
    transform, _ := world.GetComponent(entity, katsu2d.CTTransform)
    
    // Cast it to the correct type to use it
    t := transform.(*katsu2d.TransformComponent)
    
    // ... do something with the transform ...
}
```

### Creating Custom Components

While Katsu2D provides several built-in components, you'll often need to create your own to manage game-specific data.

#### 1. Defining a Component

A component is simply a Go struct that holds data. It's best practice to keep components as plain data objects, without any logic or methods.

For example, if you wanted to create a component to manage player-specific attributes, you could define it like this:

```go
// in file mygame/components/player.go
package components

// PlayerComponent holds data specific to the player entity.
type PlayerComponent struct {
    Health int
    Score  int
    Speed  float64
}
```

#### 2. Registering the Component

Before you can use a custom component, you must register it with the Katsu2D engine. This assigns a unique `ComponentID` to your component type, which is necessary for the ECS to manage it efficiently.

Registration should be done once at startup. The recommended way is to use an `init()` function in a centralized components package within your game.

```go
// in file mygame/components/components.go
package components

import (
    "github.com/edwinsyarief/katsu2d"
)

var (
    // CTPlayer will hold the ComponentID for our custom component.
    CTPlayer katsu2d.ComponentID
)

func init() {
    // Register the component and store its ID.
    // The component type is passed as a type parameter.
    CTPlayer = katsu2d.RegisterComponent[*PlayerComponent]()
}
```

Once registered, you can use `components.CTPlayer` just like you would with a built-in component ID. You can add it to entities, query for it in systems, and manage your game logic in a clean, data-oriented way.

## Why Katsu2D?

The name **Katsu2D** draws a playful and fitting analogy to the beloved Japanese dish (カツ), often a perfectly breaded and fried cutlet.

Just as a great katsu is known for its:

- **Crispy, High-Performance Exterior:** The framework aims for **blazing fast performance** and a **sharp, responsive feel** in your 2D games, much like the satisfying crunch of a well-fried panko crust.
- **Tender, Robust Interior:** Beneath the surface, Katsu2D offers a **robust and reliable ECS core**, providing a solid foundation for complex game logic and interactions, akin to the tender, flavorful meat within.
- **Simple, Satisfying Experience:** The goal is to provide a **streamlined and enjoyable development experience**, allowing you to focus on the creative aspects of your game without getting bogged down in boilerplate, just as a perfectly prepared katsu offers a simple yet deeply satisfying meal.

Combined with "2D" to clearly define its domain, **Katsu2D** signifies a framework designed to help you create games that are both **performant and a pleasure to build and play**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
