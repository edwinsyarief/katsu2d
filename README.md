# Katsu2D (カツ2D)

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue?logo=go&style=flat&logoColor=white)](https://go.dev/)
[![Ebitengine](https://img.shields.io/badge/Ebitengine-v2.10.6-db5620?style=flat)](https://ebitengine.org/)
[![Teishoku ECS](https://img.shields.io/badge/Teishoku-v1.8.0-brightgreen?style=flat)](https://github.com/edwinsyarief/teishoku)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

**Katsu2D** is a high-performance 2D game framework for Go built on [Ebitengine](https://ebitengine.org) and powered by the [Teishoku v1.8.0](https://github.com/edwinsyarief/teishoku) archetype Entity Component System (ECS). It pairs a cache-friendly, zero-allocation ECS core with a texture- and blend-batched renderer, rich gameplay systems, and environmental VFX tools designed for shipping real 2D games.

---

## Features

- **Teishoku v1.8.0 Archetype ECS**: Monomorphized `Builder`..`Builder8` handles, `Filter`..`Filter8` queries with `EachChunk` / `Chunks` / `Query` iteration, `Without` exclusions, cached `ResourceHandle[T]`, and typed `EventBus` / `EventChannel[T]`.
- **Batched GPU Renderer (`BatchRenderer`)**: Groups quads, custom meshes, and triangle strips by texture and `ebiten.Blend` mode into minimal `DrawTriangles` submissions (up to 65,532 vertices per batch).
- **Engine & Scene Management**: Functional options API, HiDPI `Layout` / `LayoutF` support, `TimeScale` simulation control, global background/overlay systems, `SceneManager` lifecycle hooks (`OnEnter`, `OnExit`, `OnUpdate`, `OnBeforeDraw`, `OnAfterDraw`), and built-in **Scene Transitions** (`Engine.TransitionTo`, `SceneManager.TransitionTo`, `FadeTransition`, `SpotlightTransition`, `LetterboxTransition`, `WipeTransition`, `CrossFadeTransition`, `SlideTransition`, automatic input blocking, unscaled time, and zero-leak pre-allocated buffers).
- **Built-in Gameplay & UI Systems**:
  - **2D Camera & Universal Frustum Culling**: `CameraSystem`, `CameraComponent` (Instant, Smooth, Spring, and DeadZone follow modes, zoom/rotation tweens, shake & bump impulses, bounds clamping, `PixelSnap` enabled by default (`true`), coordinate transforms, and `ViewBounds` culling), `ActiveCamera` (`ViewBounds`, `IsVisible`, `IsPointVisible` culling across Sprite, OrderedSprite, Shadow, Particle, Grass, Foliage, Cloth, DualGrid, and World systems), and automatic world-vs-overlay separation.
  - **Shadows & Reflections**: `ShadowRenderSystem`, `ShadowComponent` (projected, drop, contact, and color-preserving reflection shadows), `ShadowLightComponent` (global sun), coarse `IsPointVisible` pre-cull, single-draw `ShadowTypeContact` batching, single-pass anti-double-darkening compositing, and 2-pass separable Gaussian blur Kage shader with configurable half-res blur downsampling (`WithShadowBlurDownsample` / `SetBlurDownsample`).
  - **Sprites & Y-Sorting**: `SpriteSystem` (Z-sorted) and `OrderedSpriteSystem` (Z + `OrderableComponent` / automatic Y-sorting with `ParticleComponent.BlendMode` preservation for Y-ordered particles), frustum-culled via `ActiveCamera.ViewBounds`, plus grid/custom `MeshComponent` support.
  - **Frame Animations**: `AnimationSystem` with `AnimLoop`, `AnimOnce`, and `AnimBoomerang` modes.
  - **Multi-Device Input**: `InputSystem` and `InputComponent` supporting keyboard, mouse, mouse wheel, gamepad buttons, analog axes, and modifier chords.
  - **Text Rendering**: `TextSystem` with cached `GoTextFace` measurement and 9-point alignment.
  - **Vector Shapes**: `ShapeRenderSystem` with `RectangleShape`, `CircleShape`, `TriangleShape`, `PolygonShape`, `PentagonShape`, and `HexagonShape` (rounded corners, per-corner color gradients, and outer strokes).
  - **Timers & Tweens**: `TimerSystem` and `TweenSystem` publishing `TimerFinishedEvent` and `TweenFinishedEvent` across 30+ easing curves.
  - **Layer Scaling & Shaders**: `LayerSystem` offscreen compositing with aspect-preserving / pixel-perfect scaling, screen-to-canvas coordinate mapping, `IgnoreCamera` HUD/overlay support, and `ScalingFilter` presets (`SF_AASamplingSoft` through `SF_CrtLottes`).
  - **Screen Overlays**: `FadeOverlaySystem` and `CinematicOverlaySystem` (movie letterbox bars and circular spotlight cutouts).
  - **Dual-Grid Autotiling**: `DualGridTileset` (`AddMaterialWithMask`, `AddMaterialSheet`, `NewDualGridMask` with `DualMaskSoft`/`DualMaskOrganic`/`DualMaskRocky` styles), `DualGridMap` (half-tile dual-corner lookup, `PaintBrush`, `PaintRect`, `FloodFill`, and RLE JSON persistence via `MarshalJSON`/`UnmarshalJSON`/`SaveJSON`/`LoadJSON`), `DualGridComponent` (with logical-grid, dual-corner, and hover overlays), `DualGridSystem` (chunk-pooled GPU baking & frustum culling), and `WorldSystem` integration (`NewDualGridTilePainter`, `SetDualGridTile`).
  - **Chunked Biome World Generator**: `WorldGenerator` and `WorldSystem` featuring deterministic multi-octave `opensimplex` biomes, camera-overlap chunk streaming, a bounded GPU chunk-image pool, Minecraft-style RLE JSON persistence/caching by `WorldName` & `Seed`, seam-aware `SetDualGridTile`, `NewDualGridTilePainter` smooth dual-grid rendering, and `BuildMinimap`.
- **Environmental & VFX Systems**:
  - **Interactive Grass**: `GrassControllerSystem` with Perlin wind maps, radial `ForceSource` interaction, traveling `StrongWindGust`s, spring-damper physics, wind-crest shading (`WithGrassWindShading`), quadtree queries, and `ActiveCamera.ViewBounds` culling.
  - **Swaying Foliage**: `FoliageSystem` driving vertex-level wind sway, OpenSimplex ripples, radial forces, gusts, and fold/tip shading (`WithFoliageShading`) on frustum-culled grid meshes.
  - **2D Cloth / Flag Physics**: `ClothSystem` and `ClothComponent` Verlet solver with pin presets (`ClothPinLeftEdge`, `ClothPinRightEdge`, `ClothPinTopEdge`, `ClothPinTopCorners`, `ClothPinCustom`), wind/gusts/radial forces, per-vertex fold shading, zero-allocation back-to-front fold depth sorting, and `ActiveCamera.ViewBounds` culling.
  - **Particles**: `ParticleEmitterSystem`, `ParticleUpdateSystem`, and frustum-culled `ParticleRenderSystem` with 5 built-in presets (`FirePreset`, `RainPreset`, `SmokePreset`, `ExplosionPreset`, `WhimsicalPreset`); particles with `OrderableComponent` are routed to `OrderedSpriteSystem` in Y/index order with `BlendMode` preserved.
  - **Lines, Ribbon Trails & Grids**: `Line`, `LineBuilder`, `RibbonTrails`, and `GridLine` with Catmull-Rom spline smoothing, point resampling, color/width gradients, joint/cap styles, and textured lines/ribbons (`LineTextureTile` with `ebiten.AddressRepeat`, `LineTextureStretch`, `SetTextureScale`, `SetTextureOffset`).
  - **Spatial Quadtree**: Pooled `Quadtree` for fast rectangular and circular spatial queries (`QueryInto`, `QueryCircleInto`).
- **Asset Management & Utilities**: `TextureManager` (with optional texture atlasing), `FontManager`, `AudioManager` (fade in/out, stereo panning, sound stacking), `ShaderManager`, embedded/encrypted [`assetpacker`](https://github.com/edwinsyarief/assetpacker) loading, and subpackages (`atlas`, `ease`, `fsm`, `logger`, `managers`, `opensimplex`).

---

## Getting Started

### Prerequisites

- **Go 1.26** or higher
- **[Ebitengine](https://ebitengine.org) v2.10.6**
- **[Teishoku](https://github.com/edwinsyarief/teishoku) v1.8.0**

### Installation

```bash
go get github.com/edwinsyarief/katsu2d
```

### Quick Start

```go
package main

import (
	"embed"
	"image/color"
	"log"

	"github.com/edwinsyarief/katsu2d"
	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets
var embeddedFS embed.FS

func main() {
	engine := katsu2d.NewEngine(
		katsu2d.WithWindowSize(1280, 720),
		katsu2d.WithWindowTitle("Katsu2D Demo"),
		katsu2d.WithWindowResizeMode(ebiten.WindowResizingModeEnabled),
		katsu2d.WithVsyncEnabled(true),
		katsu2d.WithClearColor(color.RGBA{R: 24, G: 24, B: 32, A: 255}),
		katsu2d.WithTextureAtlas(true),
	)
	engine.InitFS(embeddedFS)

	heroTexID := engine.TextureManager().LoadEmbedded("assets/hero.png")

	scene := katsu2d.NewScene()
	scene.AddSystem(katsu2d.NewInputSystem())
	scene.AddSystem(katsu2d.NewSpriteSystem())

	var hero teishoku.Entity
	var heroAccess *teishoku.Builder2[katsu2d.TransformComponent, katsu2d.InputComponent]

	scene.OnEnter = func(e *katsu2d.Engine) {
		w := scene.World()

		// Spawn entities with Teishoku's typed Builders (1..8 components).
		spawn := teishoku.NewBuilder3[
			katsu2d.TransformComponent,
			katsu2d.SpriteComponent,
			katsu2d.InputComponent,
		](w)

		tr := katsu2d.NewTransformComponent()
		tr.Position = katsu2d.Point{X: 640, Y: 360}

		sp := katsu2d.NewSpriteComponent(heroTexID, 64, 64)

		inp := katsu2d.NewInputComponent(nil)
		katsu2d.Bind(&inp, "move_right", ebiten.KeyD)
		katsu2d.Bind(&inp, "move_left", ebiten.KeyA)

		hero = spawn.NewEntityWith(tr, sp, inp)
		heroAccess = teishoku.NewBuilder2[katsu2d.TransformComponent, katsu2d.InputComponent](w)
	}

	scene.OnUpdate = func(dt float64) {
		tr, inp := heroAccess.Get(hero)
		if tr == nil || inp == nil {
			return
		}
		const speed = 240.0
		if inp.IsPressed("move_right") {
			tr.Position.X += speed * dt
		}
		if inp.IsPressed("move_left") {
			tr.Position.X -= speed * dt
		}
	}

	engine.AddScene("main", scene)
	engine.SwitchScene("main")

	if err := engine.Run(); err != nil {
		log.Fatal(err)
	}
}
```

### Writing Custom Systems with Teishoku v1.8.0

Katsu2D systems implement `UpdateSystem` and/or `DrawSystem`. Cache Teishoku filters, `ResourceHandle[T]`, and `EventChannel[T]` inside `Initialize(w *teishoku.World)` with an `initialized` guard, and iterate contiguous component slices via `EachChunk`:

```go
type VelocityComponent struct {
	VX, VY float64
}

type MoveSystem struct {
	filter      *teishoku.Filter2[katsu2d.TransformComponent, VelocityComponent]
	initialized bool
}

func (s *MoveSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter2[katsu2d.TransformComponent, VelocityComponent](w)
	s.initialized = true
}

func (s *MoveSystem) Update(w *teishoku.World, dt float64) {
	s.filter.EachChunk(func(transforms []katsu2d.TransformComponent, vels []VelocityComponent) {
		for i := range transforms {
			transforms[i].Position.X += vels[i].VX * dt
			transforms[i].Position.Y += vels[i].VY * dt
		}
	})
}
```

You can also access world resources and typed events anywhere using:

- `teishoku.UseResource[T](w.Resources())` (or `katsu2d.GetTextureManager(w)`, `GetFontManager(w)`, `GetAudioManager(w)`, `GetShaderManager(w)`, `GetSceneManager(w)`, `GetHiResDisplayInfo(w)`).
- `teishoku.NewEventChannel[T](katsu2d.GetEventBus(w))` or `katsu2d.Subscribe[T](w, fn)` / `katsu2d.Publish[T](w, event)`.

---

## Engine & Scene Management

### Engine Functional Options

`katsu2d.NewEngine(opts ...Option)` and `katsu2d.NewEngineWithInitialCapacity(capacity, opts...)` accept the following options:

| Option | Description |
| :--- | :--- |
| `WithWindowSize(w, h int)` | Sets the initial window dimensions (default `800x600`). |
| `WithWindowTitle(title string)` | Sets the window title (default `"Game"`). |
| `WithWindowResizeMode(mode ebiten.WindowResizingModeType)` | Sets the Ebitengine window resizing mode. |
| `WithFullScreen(full bool)` | Enables or disables fullscreen mode. |
| `WithVsyncEnabled(vsync bool)` | Enables or disables vertical synchronization. |
| `WithCursorMode(mode ebiten.CursorModeType)` | Sets cursor visibility/capture mode. |
| `WithClearScreenEachFrame(clear bool)` | Controls whether Ebitengine clears the screen each frame (default `true`). |
| `WithClearColor(col color.Color)` | Sets a custom clear color applied at the start of `Draw`. |
| `WithTextureAtlas(enabled bool)` | Enables or disables the internal texture atlas packer. |
| `WithTextureAtlasSize(w, h int)` | Sets atlas dimensions (default `2048x2048`) and enables atlasing. |
| `WithUpdateSystem(sys UpdateSystem)` | Registers a global `UpdateSystem` that runs before the active scene. |
| `WithBackgroundSystem(sys any)` | Registers a global system that updates/draws **before** the active scene. |
| `WithOverlaySystem(sys any)` | Registers a global system that updates/draws **after** the active scene. |

### Scenes & Lifecycle

Create isolated scenes with `katsu2d.NewScene()` (each with its own `*teishoku.World`), register them via `engine.AddScene(name, scene)`, and switch immediately with `engine.SwitchScene(name)` or smoothly via `engine.TransitionTo(name, cfg)`.

- **Lifecycle hooks**: `OnEnter func(*Engine)`, `OnExit func(*Engine)`, `OnUpdate func(float64)`, `OnBeforeDraw func(*ebiten.Image)`, and `OnAfterDraw func(*ebiten.Image)`.
- **Global simulation**: Adjust speed dynamically with `engine.SetTimeScale(ts)` and query HiDPI backing dimensions with `engine.HiResSize()`.

### Built-in Scene Transitions

`Engine.TransitionTo(name, cfg)` and `SceneManager.TransitionTo(name, cfg)` orchestrate smooth, zero-allocation scene transitions using pre-allocated geometry/option buffers and lazily allocated reusable offscreen targets (released on teardown via `Engine.Dispose()` / `SceneManager.Dispose()`):

- **6 Built-in Transition Presets**:
  - `FadeTransition(duration, col, opts...)` — two-phase full-screen color fade (`TransitionKindFade`)
  - `SpotlightTransition(duration, col, opts...)` — two-phase circular spotlight cutout (`TransitionKindSpotlight`)
  - `LetterboxTransition(duration, col, opts...)` — two-phase horizontal shutter bars (`TransitionKindLetterbox`)
  - `WipeTransition(duration, dir, col, opts...)` — two-phase directional color curtain (`TransitionKindWipe`)
  - `CrossFadeTransition(duration, opts...)` — single-phase blend between outgoing snapshot and live incoming scene (`TransitionKindCrossFade`)
  - `SlideTransition(duration, dir, opts...)` — single-phase slide moving the outgoing snapshot off-screen (`TransitionKindSlide`)
- **Directions (`TransitionDirection`)**: `TransitionDirLeft`, `TransitionDirRight`, `TransitionDirUp`, `TransitionDirDown` (plus `TransitionWipe*` and `TransitionSlide*` aliases).
- **Functional Options (`TransitionOption`)**: `WithTransitionEase`, `WithTransitionHold`, `WithTransitionCenter`, `WithTransitionBlockInput` (automatic `InputSystem` input blocking while active, default `true`), `WithTransitionUnscaledTime` (advances on unscaled frame time independent of `TimeScale`, default `true`), `WithTransitionOnMidpoint`, `WithTransitionOnComplete`, and `WithTransitionBelowOverlay`.
- **Control Helpers**: Query `engine.IsTransitioning()` / `sm.IsTransitioning()` and abort mid-flight with `engine.CancelTransition()` / `sm.CancelTransition()`.

---

## Batched Renderer (`BatchRenderer`)

Katsu2D shares a `BatchRenderer` across systems (`engine.Renderer()`). Draw calls accumulate in pre-allocated vertex/index buffers and flush automatically whenever the active `*ebiten.Image`, `ebiten.Blend`, or the 65,532-vertex batch limit is reached:

- `AddQuad` / `AddQuadWithBlend` — transformed sprites and particles (`Scale -> Offset -> Rotate -> Translate(pos - origin)`).
- `AddCustomMeshes` / `AddCustomMeshesWithBlend` — deformed sprite grids, procedural shapes, and overlays.
- `AddTriangleStrip` — ribbon and strip geometry.
- `SetViewMatrix` / `ResetViewMatrix` — applies or clears the active camera transform (managed automatically by `Engine` and `LayerSystem`).
- `Begin(screen)` / `Flush()` / `GetScreen()` — manual batch control when interoperating with direct Ebitengine draw calls.

---

## Built-in Systems & Components

| Feature | Systems | Components & Helpers |
| :--- | :--- | :--- |
| **Scene Transitions** | `SceneManager` (`Engine.TransitionTo`) | `TransitionConfig`, `FadeTransition`, `SpotlightTransition`, `LetterboxTransition`, `WipeTransition`, `CrossFadeTransition`, `SlideTransition`, `TransitionOption` |
| **Camera** | `CameraSystem` | `CameraComponent` (`NewCameraComponent`, `PixelSnap` default `true`, `CameraFollowInstant`/`Smooth`/`Spring`/`DeadZone`), `ActiveCamera` (`GetActiveCamera`, `ViewBounds`, `IsVisible`, `IsPointVisible`) |
| **Shadows** | `ShadowRenderSystem` | `ShadowComponent` (`NewProjectedShadow`, `NewDropShadow`, `NewContactShadow`, `NewReflectionShadow`), `ShadowLightComponent`, `WithShadowBlurDownsample`, `WithShadowFrustumMargin` |
| **Sprites & Y-Sorting** | `SpriteSystem`, `OrderedSpriteSystem` | `TransformComponent`, `SpriteComponent`, `MeshComponent` (`NewGridMeshComponent`), `OrderableComponent` (`NewOrderableComponent`, `NewYOrderableComponent`), Y-ordered `ParticleComponent` blend preservation, `WithSpriteFrustumMargin`, `WithOrderedSpriteFrustumMargin` |
| **Animations** | `AnimationSystem` | `AnimationComponent` (`AnimLoop`, `AnimOnce`, `AnimBoomerang`) |
| **Input** | `InputSystem` | `InputComponent`, `Bind`, `BatchBind`, `NewKeyConfig`, `IsPressed`, `IsJustPressed`, `IsJustReleased` |
| **Text** | `TextSystem` | `TextComponent`, `NewTextComponent`, `TextAlignment` (9 anchor points) |
| **Vector Shapes** | `ShapeRenderSystem` | `ShapeComponent`, `Shape` interface, `RectangleShape`, `CircleShape`, `TriangleShape`, `PolygonShape`, `PentagonShape`, `HexagonShape` |
| **Timers & Tweens** | `TimerSystem`, `TweenSystem` | `TimerComponent` (`TimerStateIdle`/`Active`/`Done`), `TweenComponent`, `EaseType` (`Linear`..`ElasticOutIn`), `TimerFinishedEvent`, `TweenFinishedEvent` |
| **Layer Scaling & Shaders** | `LayerSystem` | `NewLayerSystem(w, h, opts...)` with `AddSystem`, `Stretched`, `PixelPerfect`, `IgnoreCamera`, `SetFilter`, `ScreenToCanvas`, `ScaleCursorPosition`, `Dispose` |
| **Overlays** | `FadeOverlaySystem`, `CinematicOverlaySystem` | `FadeOverlayComponent` (`SpawnFadeOverlay`, `FadeTypeIn`/`Out`), `CinematicOverlayComponent` (`SpawnCinematicOverlay`, `EndCinematic`, `CinematicMovie`, `CinematicSpotlight`) |
| **2D Cloth Physics** | `ClothSystem` | `ClothComponent` (`NewClothComponent`, `ClothPinLeftEdge`/`RightEdge`/`TopEdge`/`TopCorners`/`Custom`, `WithClothWind`, `WithClothShading`, `SetWind`, `Reset`), zero-alloc fold depth sorting |
| **Dual-Grid Autotiling** | `DualGridSystem` | `DualGridComponent` (`NewDualGridComponent`), `DualGridTileset` (`NewDualGridTileset`, `NewDualGridMask`), `DualGridMap` (`NewDualGridMap`, `PaintBrush`, `PaintRect`, `FloodFill`, `SaveJSON`, `LoadJSON`), `NewDualGridTilePainter`, `SetDualGridTile` |
| **Chunked World** | `WorldSystem` | `WorldGenerator` (`NewWorldGenerator`, `WorldGenConfig`, `BiomeDef`, `WorldChunk`, `SetDualGridTile`, `BuildMinimap`, `Save`, `Regenerate`, `WorldStreamStats`), `NewDualGridTilePainter` |

### Layer Scaling Filters (`ScalingFilter`)

`LayerSystem.SetFilter(filter)` supports the following scaling filters:

- `SF_AASamplingSoft` — anti-aliased pixel-art point sampling (balanced sharpness and zoom stability)
- `SF_AASamplingSharp` — sharper anti-aliased pixel-art sampling
- `SF_Nearest` — crisp nearest-neighbor scaling
- `SF_Hermite` — Hermite interpolation
- `SF_Bicubic` — smooth bicubic interpolation
- `SF_Bilinear` — smooth bilinear interpolation
- `SF_CrtLottes` — Lottes CRT shader filter

---

## Camera

Katsu2D provides a pure-data ECS 2D camera via `CameraComponent`, `CameraSystem` (`NewCameraSystem()`), and the `ActiveCamera` world resource (`GetActiveCamera(w)`):

- **4 Target Follow Modes** (`FollowEntity`, `FollowEntityImmediate`, `SnapToTarget`, `StopFollowing`):
  - `CameraFollowInstant` — snaps directly to the target + `TargetOffset`.
  - `CameraFollowSmooth` — frame-rate-independent exponential smoothing (`FollowSpeed`).
  - `CameraFollowSpring` — damped harmonic spring (`SpringStiffness`, `SpringDamping`, `Velocity`).
  - `CameraFollowDeadZone` — normalized viewport `DeadZone` with velocity, `TrackingFriction`, and `EdgeBrakeDistance` braking near world bounds.
- **Tweens, Zoom & Rotation**: `MoveTo`, `ZoomTo`, and `RotateTo` with any `EaseType`, `BumpZoom`, clamped zoom limits (`MinZoom`, `MaxZoom`, `WithCameraZoomLimits`), and `CameraComponent.PixelSnap` (enabled by default, `true`, toggleable via `WithCameraPixelSnap`).
- **Screen Shake & Bump**: `Shake(duration, power)`, `ShakeWithRotation(duration, power, rotPower)`, `Bump(offset)`, and `BumpAngular(angle, distance)` with exponential decay.
- **Bounds Clamping**: `SetBounds` / `WithCameraBounds` and `ClearBounds` keep the camera viewport within world limits.
- **Coordinates & Universal Frustum Culling**: `WorldToScreen`, `ScreenToWorld`, `ViewBounds`, `IsVisible(bounds)`, and `IsPointVisible(point, margin)` on both `CameraComponent` and `ActiveCamera`. Built-in render and simulation systems (`SpriteSystem`, `OrderedSpriteSystem`, `ShadowRenderSystem`, `ParticleRenderSystem`, `GrassControllerSystem`, `FoliageSystem`, `ClothSystem`, `DualGridSystem`, and `WorldSystem`) automatically cull against `ActiveCamera.ViewBounds` when an active camera is present.
- **Automatic World-vs-Overlay Separation**: `Engine.Draw` automatically applies the active camera's `ViewMatrix` via `BatchRenderer.SetViewMatrix` for background systems and the active scene, then calls `ResetViewMatrix()` before `overlayDrawSystems` so HUD/UI overlays stay in screen space. `LayerSystem` also applies `ActiveCamera.ViewMatrix` by default unless created with `IgnoreCamera(true)`.

```go
scene.AddSystem(katsu2d.NewCameraSystem())
scene.AddSystem(katsu2d.NewSpriteSystem())

// Spawn a camera following the hero inside world bounds
cam := katsu2d.NewCameraComponent(
	1280, 720,
	katsu2d.WithCameraFollowMode(katsu2d.CameraFollowDeadZone),
	katsu2d.WithCameraDeadZone(katsu2d.V(0.04, 0.10)),
	katsu2d.WithCameraBounds(katsu2d.NewRectangle(0, 0, 3200, 1800)),
	katsu2d.WithCameraPixelSnap(true),
)
cam.FollowEntity(hero, katsu2d.V(0, -16))
cam.ZoomTo(1.5, 0.4, katsu2d.CubicOut)
cam.Shake(0.25, 8.0)

teishoku.NewBuilder[katsu2d.CameraComponent](w).NewEntityWith(*cam)

// Query ActiveCamera anywhere for coordinate conversion and frustum culling
if active := katsu2d.GetActiveCamera(w); active != nil {
	worldPos := active.ScreenToWorld(katsu2d.V(mx, my))
	_ = active.IsVisible(entityBounds)
	_ = worldPos
}

// Keep HUD layers pinned to screen space
hudLayer := katsu2d.NewLayerSystem(1280, 720,
	katsu2d.IgnoreCamera(true),
	katsu2d.AddSystem(katsu2d.NewTextSystem()),
)
```

---

## Shadows

Katsu2D renders 2D shadows and reflections with `ShadowComponent`, optional `ShadowLightComponent` global sunlight, and `ShadowRenderSystem`:

- **Shadow Types** (`ShadowComponent` — zero-pointer POD):
  - `ShadowTypeProjected` (`NewProjectedShadow(angle, length)`) — directional skewed silhouette projection anchored at the caster's base, with optional tip fade-out (`FadeOut`, `FadeAlpha`) and `Elevation`.
  - `ShadowTypeDrop` (`NewDropShadow(offset)`) — offset silhouette copy of the caster's sprite.
  - `ShadowTypeContact` (`NewContactShadow(radiusX, radiusY)`) — soft elliptical ground contact blob using a shared radial gradient texture.
  - `NewReflectionShadow(length)` — color-preserving (`PreserveColor: true`) vertical reflection for water or polished floors.
- **Global Sun (`ShadowLightComponent`)**: Spawn `NewShadowLightComponent(angle, length, opacity)` once in the world to drive all `ShadowComponent`s with `UseGlobalLight: true`.
- **`ShadowRenderSystem` (`NewShadowRenderSystem(opts...)`)**:
  - **Direct Batched Mode** (default): appends shadow quads directly into the shared `BatchRenderer` in a single batched pass.
  - **Single-Pass Composited Mode** (`WithShadowComposited(true)` / `SetComposited(true)`): renders non-color-preserving shadows into an offscreen mask and composites once with uniform opacity/tint (`WithShadowGlobalOpacity`, `WithShadowGlobalColor`) to prevent overlapping double-darkening.
  - **2-Pass Separable Gaussian Blur** (`WithShadowBlur(radius)` / `SetBlurRadius(radius)`): applies a fast 2-pass separable Kage blur shader across the offscreen mask (implicitly enables composited mode), using half-resolution blur downsampling by default (`WithShadowBlurDownsample(scale)` / `SetBlurDownsample(scale)`, default `0.5`, clamped to `[0.25, 1.0]`).
  - **10K Optimization & Frustum Culling**: performs an early coarse `IsPointVisible` pre-cull against `ActiveCamera.ViewBounds` expanded by `WithShadowFrustumMargin(margin)` (default `128`) and batches `ShadowTypeContact` quads into single vertex/index submissions before flushing. Call `Dispose()` on teardown to release offscreen GPU render targets.

```go
shadows := katsu2d.NewShadowRenderSystem(
	katsu2d.WithShadowComposited(true),
	katsu2d.WithShadowBlur(3.0),
	katsu2d.WithShadowGlobalOpacity(0.45),
)
scene.AddSystem(shadows) // Register before SpriteSystem so shadows draw underneath
scene.AddSystem(katsu2d.NewSpriteSystem())

// Global directional sun
teishoku.NewBuilder[katsu2d.ShadowLightComponent](w).NewEntityWith(
	katsu2d.NewShadowLightComponent(0.6, 0.5, 0.45),
)

// Attach a projected shadow (or NewDropShadow / NewContactShadow / NewReflectionShadow)
teishoku.NewBuilder3[
	katsu2d.TransformComponent,
	katsu2d.SpriteComponent,
	katsu2d.ShadowComponent,
](w).NewEntityWith(tr, sp, katsu2d.NewProjectedShadow(0.6, 0.5))

// Release offscreen GPU targets when tearing down the scene
scene.OnExit = func(*katsu2d.Engine) {
	shadows.Dispose()
}
```

---

## Environmental & VFX Systems

### Grass (`GrassControllerComponent`, `GrassControllerSystem`)

Simulates fields of individual grass blades (`GrassComponent`) with a precomputed Perlin wind map, spring-damper recovery physics, radial external forces (`ForceSource` via `SetForcePositions`), traveling wind gusts (`StrongWindGust` via `AddStrongWindGust`), sunlit wind-wave crest shading (`WithGrassWindShading`), `ActiveCamera.ViewBounds` culling, and spatial `Quadtree` queries.

Configure `NewGrassControllerComponent(world, tm, worldW, worldH, textureID, z, opts...)` with `GrassOption`s:
`WithGrassTileSize`, `WithGrassDensity`, `WithGrassWindForce`, `WithGrassWindSpeed`, `WithGrassWindDirection`, `WithGrassWindShading`, `WithGrassNoiseMapSize`, `WithGrassNoiseFrequency`, `WithGrassForceBaseAcceleration`, `WithGrassSwaySpringStrength`, `WithGrassSwayDamping`, `WithGrassAreas` (`[]Area`), and `WithGrassOrderable`. Call `Dispose()` (or `DisposeEntities(w)`) when tearing down a field.

### Foliage (`FoliageComponent`, `FoliageControllerComponent`, `FoliageSystem`)

Deforms grid `MeshComponent` vertices on sprites (`SpriteComponent` + `MeshComponent` + `FoliageComponent`) using OpenSimplex noise for organic canopy sway, directional ripples, radial external forces (`SetForcePositions`), traveling wind gusts (`AddStrongWindGust`), base-to-tip / wind-crest vertex shading (`WithFoliageShading`), and `ActiveCamera.ViewBounds` culling.

Configure `NewFoliageControllerComponent(opts...)` with `WithFoliageWindForce`, `WithFoliageWindSpeed`, `WithFoliageWindDirection`, `WithFoliageRippleStrength`, and `WithFoliageShading`, and attach `NewFoliageComponent(swaySeed, pivot)` to foliage entities.

### 2D Cloth & Flag Physics (`ClothComponent`, `ClothSystem`)

Simulates 2D Verlet cloth grids paired with `TransformComponent`, `SpriteComponent`, and `MeshComponent` (`MeshTypeGrid`), supporting frame-rate-independent damping, iterative constraint relaxation, ambient traveling flutter waves, radial force interaction (`ClothSystem.SetForceSources` / `ClothComponent.ExternalForces`), traveling wind gusts (`ClothSystem.AddWindGust`), per-vertex fold shading (`WithClothShading`), zero-allocation back-to-front quad fold depth sorting (in-place insertion sort over pre-allocated quad buffers), and `ActiveCamera.ViewBounds` culling.

- **Pin Presets (`ClothPinMode`)**: `ClothPinLeftEdge`, `ClothPinRightEdge`, `ClothPinTopEdge`, `ClothPinTopCorners`, and `ClothPinCustom` (via `WithClothPinFunc`).
- **Configuration (`NewClothComponent(cols, rows, opts...)`)**: `WithClothPinMode`, `WithClothPinFunc`, `WithClothGravity`, `WithClothWind`, `WithClothDamping`, `WithClothStiffness`, `WithClothShading`, and `WithClothPhase`, plus runtime `SetWind` and `Reset`.

### Particles (`ParticleEmitterSystem`, `ParticleUpdateSystem`, `ParticleRenderSystem`)

Each spawned particle is a self-contained POD `ParticleComponent` entity supporting gravity, zigzag or OpenSimplex noise motion (`ParticleDirectionMode`), fade curves (`ParticleFadeMode`), scale curves (`ParticleScaleMode`), rotation/spin, custom `ebiten.Blend` modes, automatic `OrderableComponent` inheritance from ordered emitters, and `ActiveCamera.ViewBounds` frustum culling in `ParticleRenderSystem`. Particles carrying `OrderableComponent` are skipped by `ParticleRenderSystem` and drawn in Y/index order by `OrderedSpriteSystem`, which preserves each particle's `ParticleComponent.BlendMode` (e.g., `ebiten.BlendLighter` for additive campfire flames) via `BatchRenderer.AddQuadWithBlend`.

Five ready-to-use presets are included:

- `FirePreset(textureID int)` — glowing additive rising flames
- `RainPreset(textureID int)` — fast downward rainfall streaks
- `SmokePreset(textureIDs []int)` — soft expanding and fading smoke puffs
- `ExplosionPreset(textureIDs []int)` — one-shot additive radial burst
- `WhimsicalPreset(textureID int)` — swirling noise-perturbed glowing particles

### Lines, Ribbon Trails & Grids

- **`Line` (`NewLine`) & `LineBuilder` (`NewLineBuilder`)**: Polyline mesh generator supporting uniform or per-point interpolated colors and widths, `LineJointSharp` / `LineJointBevel` / `LineJointRound`, `LineCapNone` / `LineCapBox` / `LineCapRound`, closed loops, texturing (`LineTextureNone`, `LineTextureTile` with automatic `ebiten.AddressRepeat`, `LineTextureStretch`, `SetTexture`, `SetTextureMode`, `SetTextureScale`, `SetTextureOffset`), Catmull-Rom spline smoothing (`EnableCatmullRom`), uniform point resampling (`SetStepDistance`), and debug mesh visualization (`SetDebugDraw`).
- **`RibbonTrails` (`NewRibbonTrails`)**: Dynamic motion trails with time-based point expiration (`SetLifetime`), point limits (`SetLimit`), width/color gradients (`SetWidths`, `SetColors`), texturing (`LineTextureTile`, `LineTextureStretch`, `SetTexture`, `SetTextureMode`, `SetTextureScale`, `SetTextureOffset`), Catmull-Rom smoothing, and step resampling.
- **`GridLine` (`NewGridLine`)**: Renders an entire `rows x cols` grid in a single triangle-mesh draw call with configurable corner joins (`GridCornerSharp`, `GridCornerBevel`, `GridCornerRound`).

### Chunked Biome World Generator (`WorldGenerator`, `WorldSystem`)

Generates, caches, and streams large multi-biome 2D worlds around `ActiveCamera.ViewBounds`:

- **`WorldGenerator` (`NewWorldGenerator(cfg WorldGenConfig)`)**: Uses multi-octave `opensimplex` FBM elevation, moisture, and temperature maps to classify tiles across registered `BiomeDef`s (or a custom `BiomeClassifier`). Supports Minecraft-style RLE-encoded JSON persistence/caching keyed by `WorldName` and `Seed` (`StorageDir`, `LoadedFromCache`, `CachePath`, `Save`, `Regenerate`), runtime tile edits (`SetTile`, seam-aware `SetDualGridTile`, `TileAt`, `BiomeAt`, `BiomeAtWorld`), and full-world pixel-art overview generation via `BuildMinimap()`.
- **`WorldSystem` (`NewWorldSystem(gen, opts...)`)**: Streams `WorldChunk`s in and out based on camera-overlap margins (`LoadMarginChunks`, `UnloadMarginChunks`), bakes tiles into a bounded GPU chunk-image pool (`MaxPooledImages`), manages chunk-owned entity lifetimes (`WithWorldOnChunkLoad`, `WithWorldOnChunkUnload`), supports custom tile baking (`WithWorldTileTextures`, `WithWorldTilePainter` including `NewDualGridTilePainter`, `WithWorldFallbackColor`), reports telemetry via `Stats()` (`WorldStreamStats`), and releases GPU targets cleanly via `UnloadAll(w)` and `Dispose()`.

### Dual-Grid Autotiling (`DualGridTileset`, `DualGridMap`, `DualGridSystem`)

Provides half-tile offset dual-grid autotiling with only 16 bitmasks per material, layered terrain transitions, chunk-pooled GPU baking, and `WorldSystem` integration:

- **Procedural Alpha Masks (`NewDualGridMask`)**: Generates seamless `4*tileSize x 4*tileSize` bitmask atlases in `DefaultDualMaskOrder` with `DualMaskSoft`, `DualMaskOrganic`, or `DualMaskRocky` edge silhouettes.
- **Material Atlas (`DualGridTileset`)**: `NewDualGridTileset(tileSize)` packs registered `DualMaterialID` layers into a single `(16*tileSize) x (numMaterials*tileSize)` GPU atlas (`BuildAtlas()`, `Atlas()`, `Dispose()`). Register materials via `AddMaterialWithMask(name, materialImg, mask4x4, opts...)` or pre-authored 4×4 sheets via `AddMaterialSheet(name, sheet4x4, opts...)` with `WithDualMaskOrder` and `WithDualMaterialColor`.
- **Editor-Ready Tile Grid & RLE JSON Persistence (`DualGridMap`)**: `NewDualGridMap(width, height, tileSize, defaultMat, WithDualGridChunkSize(cs))` manages logical tiles, automatically dirtying affected chunk seams on edits (`SetTile`, `Fill`, `PaintBrush`, `PaintRect`, `FloodFill`), provides exact coordinate helpers (`WorldToTile`, `TileToWorld`, `TileAt`, `WorldPixelSize`), and serializes compact RLE-compressed JSON via `MarshalJSON`, `UnmarshalJSON`, `SaveJSON`, and `LoadJSON`.
- **ECS Rendering & Chunk Pooling (`DualGridComponent`, `DualGridSystem`)**: Attach `NewDualGridComponent(gridMap, tileset)` (with optional `ShowLogicalGrid`, `ShowDualCorners`, and `HasHover`/`HoverX`/`HoverY` editor overlays) and register `NewDualGridSystem(opts...)` (`WithDualGridMaxPooledImages`, `WithDualGridLoadMargin`, `WithDualGridUnloadMargin`, `Stats()`, `Dispose()`).
- **World Streaming Integration (`NewDualGridTilePainter`, `SetDualGridTile`)**: Pass `NewDualGridTilePainter(gen, tileset, mapTile)` to `WithWorldTilePainter(...)` on `WorldSystem` for single-batch smooth dual-grid chunk baking, and edit tiles across chunk borders with `WorldGenerator.SetDualGridTile(tx, ty, tileID)`.

### Spatial Quadtree (`Quadtree`)

`NewQuadtree(world, bounds)` indexes entities with `TransformComponent` using pooled nodes (`sync.Pool`), supporting zero-allocation slice reuse via `QueryInto(bounds, dst)` and `QueryCircleInto(center, radius, dst)` as well as `Query`, `QueryCircle`, `Insert`, and `Clear`.

---

## Asset Management & Subpackages

### Asset Managers

Load assets at startup through the `Engine` managers (and retrieve them inside systems via `GetTextureManager(w)`, `GetFontManager(w)`, `GetAudioManager(w)`, `GetShaderManager(w)`):

- **`TextureManager`** — loads textures from disk, `embed.FS` (`LoadEmbedded`), or encrypted `assetpacker` bundles, with optional automatic atlas packing (`WithTextureAtlas`).
- **`FontManager`** — loads and caches font sources for `TextSystem`.
- **`AudioManager`** — manages audio playback with fade in/out, stereo panning, and sound stacking.
- **`ShaderManager`** — loads and manages Kage shaders.
- **Asset Sources**: Call `engine.InitFS(embedFS)` for Go `embed.FS` assets or `engine.InitAssetReader(path, key)` for encrypted `assetpacker` bundles.

```go
var (
	LogoTextureID int
	DefaultFontID int
)

func loadAssets(e *katsu2d.Engine) {
	LogoTextureID = e.TextureManager().LoadEmbedded("embedded/images/ebitengine_logo.png")
	DefaultFontID = e.FontManager().LoadEmbedded("embedded/fonts/default.ttf")
}
```

### Subpackages

| Subpackage | Purpose |
| :--- | :--- |
| `atlas` | Standalone texture atlas (`Atlas`, `Set`, `Image`) for rectangular packing on unmanaged/managed Ebitengine images. |
| `ease` | Generic easing functions (`EaseFunc[T]`) covering linear, quad, cubic, quart, quint, sine, expo, circ, back, bounce, and elastic curves. |
| `fsm` | Flyweight generic finite state machine (`StateMachineConfig[S, E]`, `StateMachine[S, E]`) with guards and `OnEnter` / `OnExit` callbacks. |
| `logger` | Cross-platform (desktop + WebAssembly) leveled logger (`Error`, `Info`, `Debug`) with atomic debug toggle. |
| `managers` | Supporting resource and manager primitives. |
| `opensimplex` | Pure-Go 2D/3D OpenSimplex noise generator (`opensimplex.New(seed)`). |

---

## Examples

For complete runnable games and feature showcases, see the [katsu2d-examples](https://github.com/edwinsyarief/katsu2d-examples) repository.

## Why Katsu2D?

The name **Katsu2D** draws a playful and fitting analogy to the beloved Japanese dish (カツ), often a perfectly breaded and fried cutlet.

Just as a great katsu is known for its:

- **Crispy, High-Performance Exterior:** The framework aims for **blazing fast performance** and a **sharp, responsive feel** in your 2D games, much like the satisfying crunch of a well-fried panko crust.
- **Tender, Robust Interior:** Beneath the surface, Katsu2D offers a **robust and reliable ECS core**, providing a solid foundation for complex game logic and interactions, akin to the tender, flavorful meat within.
- **Simple, Satisfying Experience:** The goal is to provide a **streamlined and enjoyable development experience**, allowing you to focus on the creative aspects of your game without getting bogged down in boilerplate, just as a perfectly prepared katsu offers a simple yet deeply satisfying meal.

Combined with "2D" to clearly define its domain, **Katsu2D** signifies a framework designed to help you create games that are both **performant and a pleasure to build and play**.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.
