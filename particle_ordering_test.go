package katsu2d

import (
	"image/color"
	"testing"

	"github.com/edwinsyarief/teishoku"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestOrderedParticleRendersInYOrderWithBlend(t *testing.T) {
	w := teishoku.NewWorld(64)
	tm := NewTextureManager()
	teishoku.AddResource(w.Resources(), tm)

	emSys := NewParticleEmitterSystem()
	emSys.Initialize(w)

	ordSys := NewOrderedSpriteSystem()
	ordSys.Initialize(w)

	pRenderSys := NewParticleRenderSystem()
	pRenderSys.Initialize(w)
	pRenderSys.Initialize(w) // idempotent guard

	treeBuilder := teishoku.NewBuilder3[TransformComponent, SpriteComponent, OrderableComponent](w)

	bgTreeTr := NewTransformComponent()
	bgTreeTr.Position = Point{X: 100, Y: 140}
	bgTreeTr.Z = 1
	bgTree := treeBuilder.NewEntityWith(
		bgTreeTr,
		NewSpriteComponent(0, 16, 16),
		OrderableComponent{Index: 140},
	)

	fgTreeTr := NewTransformComponent()
	fgTreeTr.Position = Point{X: 100, Y: 260}
	fgTreeTr.Z = 1
	fgTree := treeBuilder.NewEntityWith(
		fgTreeTr,
		NewSpriteComponent(0, 16, 16),
		OrderableComponent{Index: 260},
	)

	emitterTr := NewTransformComponent()
	emitterTr.Position = Point{X: 100, Y: 200}
	emitterTr.Z = 1
	teishoku.NewBuilder3[TransformComponent, ParticleEmitterComponent, OrderableComponent](w).NewEntityWith(
		emitterTr,
		*FirePreset(0),
		OrderableComponent{Index: 200},
	)

	// Step emitter to spawn campfire particles (EmitRate=200 * dt=0.05 -> 10 particles).
	emSys.Update(w, 0.05)

	particleFilter := teishoku.NewFilter3[TransformComponent, SpriteComponent, ParticleComponent](w)
	orderedParticleFilter := teishoku.NewFilter2[ParticleComponent, OrderableComponent](w)

	campfireCount := orderedParticleFilter.Count()
	if campfireCount == 0 {
		t.Fatal("expected campfire emitter to spawn ordered particles")
	}

	orderedParticleFilter.EachChunk(func(particles []ParticleComponent, orderables []OrderableComponent) {
		for i := range particles {
			if orderables[i].Index != 200 {
				t.Fatalf("expected spawned campfire particle OrderableComponent.Index == 200, got %v", orderables[i].Index)
			}
			if particles[i].BlendMode != ebiten.BlendLighter {
				t.Fatalf("expected campfire particle BlendLighter, got %+v", particles[i].BlendMode)
			}
		}
	})

	// Spawn an unordered rain particle (no OrderableComponent).
	rainTr := NewTransformComponent()
	rainTr.Position = Point{X: 50, Y: 50}
	rainTr.Z = 1
	rainEnt := teishoku.NewBuilder3[TransformComponent, SpriteComponent, ParticleComponent](w).NewEntityWith(
		rainTr,
		NewSpriteComponent(0, 4, 8),
		ParticleComponent{
			Lifetime:      1.0,
			TotalLifetime: 1.0,
			InitialColor:  color.RGBA{100, 150, 255, 200},
			TargetColor:   color.RGBA{100, 150, 255, 0},
			BlendMode:     ebiten.BlendSourceOver,
		},
	)

	if got := particleFilter.Count(); got != campfireCount+1 {
		t.Fatalf("expected %d total particles (%d campfire + 1 rain), got %d", campfireCount+1, campfireCount, got)
	}

	// Update OrderedSpriteSystem and verify sort order: bgTree -> campfire particles -> fgTree.
	ordSys.Update(w, 1.0/60.0)

	wantEntries := 2 + campfireCount
	if len(ordSys.entries) != wantEntries {
		t.Fatalf("expected %d ordered sprite entries, got %d", wantEntries, len(ordSys.entries))
	}
	if ordSys.entries[0].entity != bgTree {
		t.Fatalf("expected background tree (%v) first, got %v", bgTree, ordSys.entries[0].entity)
	}
	for i := 1; i <= campfireCount; i++ {
		e := ordSys.entries[i].entity
		if e == bgTree || e == fgTree || e == rainEnt {
			t.Fatalf("expected campfire particle at entry %d, got %v", i, e)
		}
		if ordSys.entries[i].order != 200 {
			t.Fatalf("expected entry %d order == 200, got %v", i, ordSys.entries[i].order)
		}
	}
	if ordSys.entries[len(ordSys.entries)-1].entity != fgTree {
		t.Fatalf("expected foreground tree (%v) last, got %v", fgTree, ordSys.entries[len(ordSys.entries)-1].entity)
	}

	// Verify OrderedSpriteSystem.Draw honors particle BlendLighter in BatchRenderer
	// and transitions back to BlendSourceOver for the foreground tree.
	screen := ebiten.NewImage(320, 240)
	defer screen.Deallocate()
	rdr := NewBatchRenderer()

	// Draw only the ordered campfire slice first to inspect the active batch blend mode directly.
	rdr.Begin(screen)
	ordSys.worldVertices = drawSpriteEntries(
		ordSys.entries[1:1+campfireCount],
		ordSys.tsBuilder,
		ordSys.meshBuilder,
		ordSys.particleBuilder,
		ordSys.tmHandle.Get(),
		ordSys.transform,
		ordSys.worldVertices,
		rdr,
	)
	if len(rdr.vertices) != campfireCount*4 {
		t.Fatalf("expected %d vertices for campfire particles, got %d", campfireCount*4, len(rdr.vertices))
	}
	if rdr.currentBlend != ebiten.BlendLighter {
		t.Fatalf("expected BatchRenderer active blend to be BlendLighter, got %+v", rdr.currentBlend)
	}

	// Full OrderedSpriteSystem.Draw submits bgTree (BlendSourceOver) -> campfire (BlendLighter) -> fgTree (BlendSourceOver).
	rdr.Begin(screen)
	ordSys.Draw(w, rdr)
	if len(rdr.vertices) != 4 {
		t.Fatalf("expected final flushed batch to contain only foreground tree (4 vertices), got %d", len(rdr.vertices))
	}
	if rdr.currentBlend != ebiten.BlendSourceOver {
		t.Fatalf("expected final batch blend to be BlendSourceOver for foreground tree, got %+v", rdr.currentBlend)
	}

	// Verify ParticleRenderSystem.Draw skips all ordered campfire particles and only draws the unordered rain particle.
	rdr.Begin(screen)
	pRenderSys.Draw(w, rdr)
	if len(rdr.vertices) != 4 {
		t.Fatalf("expected ParticleRenderSystem to submit only 1 unordered rain particle (4 vertices), got %d", len(rdr.vertices))
	}
	if rdr.currentBlend != ebiten.BlendSourceOver {
		t.Fatalf("expected unordered rain particle blend BlendSourceOver, got %+v", rdr.currentBlend)
	}
}
