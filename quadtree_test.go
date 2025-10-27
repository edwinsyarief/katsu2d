package katsu2d

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/mlange-42/ark/ecs"
)

// Test constants
const (
	testWorldSize  = 1000.0 // World bounds size for tests
	testInitialCap = 10000  // Initial entity capacity for world
)

// Helper function to create a test world and quadtree.
func setupTestQuadtree() (*ecs.World, *Quadtree, Rectangle) {
	world := ecs.NewWorld(testInitialCap)
	bounds := Rectangle{
		Min: Vector{X: 0, Y: 0},
		Max: Vector{X: testWorldSize, Y: testWorldSize},
	}
	qt := NewQuadtree(&world, bounds)
	return &world, qt, bounds
}

// TestNewQuadtree verifies that a new quadtree is properly initialized.
func TestNewQuadtree(t *testing.T) {
	world, qt, bounds := setupTestQuadtree()
	if qt.root == nil {
		t.Fatal("Root node should not be nil")
	}
	if qt.root.bounds != bounds {
		t.Errorf("Expected bounds %+v, got %+v", bounds, qt.root.bounds)
	}
	if qt.root.depth != 0 {
		t.Errorf("Expected depth 0, got %d", qt.root.depth)
	}
	if len(qt.root.objects) != 0 {
		t.Errorf("Expected 0 objects, got %d", len(qt.root.objects))
	}
	if qt.root.children[0] != nil || qt.root.children[1] != nil ||
		qt.root.children[2] != nil || qt.root.children[3] != nil {
		t.Error("Children should be nil initially")
	}
	if qt.world != world {
		t.Error("World reference mismatch")
	}
}

// TestInsertBasic tests inserting entities within bounds.
func TestInsertBasic(t *testing.T) {
	world, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](world)

	// Create and insert entities
	numEntities := MaxObjectsPerNode - 1 // Less than max to avoid subdivision
	entities := make([]ecs.Entity, numEntities)
	for i := 0; i < numEntities; i++ {
		ent := builder.NewEntity(&TransformComponent{Position: Point{X: 100 + float64(i), Y: 100 + float64(i)}})
		qt.Insert(ent)
		entities[i] = ent
	}

	// Check root node
	if len(qt.root.objects) != numEntities {
		t.Errorf("Expected %d objects in root, got %d", numEntities, len(qt.root.objects))
	}
	for _, ent := range entities {
		found := false
		for _, o := range qt.root.objects {
			if o == ent {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Entity %v not found in root", ent)
		}
	}
	if qt.root.children[0] != nil {
		t.Error("No subdivision expected")
	}
}

// TestInsertOutsideBounds tests inserting an entity outside the quadtree bounds.
func TestInsertOutsideBounds(t *testing.T) {
	_, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](qt.world)

	ent := builder.NewEntity(&TransformComponent{Position: Point{X: -100, Y: -100}})
	qt.Insert(ent)

	if len(qt.root.objects) != 0 {
		t.Errorf("Expected 0 objects, got %d", len(qt.root.objects))
	}
}

// TestSubdivision tests that the quadtree subdivides when exceeding max objects.
func TestSubdivision(t *testing.T) {
	world, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](world)

	// Insert MaxObjectsPerNode + 1 entities in the same quadrant (top-left)
	numEntities := MaxObjectsPerNode + 1
	entities := make([]ecs.Entity, numEntities)
	for i := 0; i < numEntities; i++ {
		ent := builder.NewEntity(&TransformComponent{Position: Point{X: 100 + float64(i), Y: 100 + float64(i)}})
		qt.Insert(ent)
		entities[i] = ent
	}

	// Root should have subdivided
	if qt.root.children[0] == nil {
		t.Fatal("Expected subdivision")
	}
	if len(qt.root.objects) != 0 {
		t.Errorf("Expected 0 objects in root after subdivision, got %d", len(qt.root.objects))
	}

	// Check that all entities are still queryable
	queried := qt.Query(qt.root.bounds)
	if len(queried) != numEntities {
		t.Errorf("Expected %d entities from query, got %d", numEntities, len(queried))
	}
}

// TestQueryRectangle tests querying entities within a rectangle.
func TestQueryRectangle(t *testing.T) {
	world, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](world)

	// Insert entities
	entities := make([]ecs.Entity, 5)
	positions := []Point{
		{100, 100}, {200, 200}, {300, 300}, {900, 900}, {950, 950},
	}
	for i := range entities {
		ent := builder.NewEntity(&TransformComponent{Position: positions[i]})
		qt.Insert(ent)
		entities[i] = ent
	}

	// Query a rectangle that contains the first three
	queryBounds := Rectangle{Min: Vector{0, 0}, Max: Vector{400, 400}}
	result := qt.Query(queryBounds)

	if len(result) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(result))
	}
}

// TestQueryCircle tests querying entities within a circle.
func TestQueryCircle(t *testing.T) {
	world, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](world)

	// Insert entities
	entities := make([]ecs.Entity, 5)
	positions := []Point{
		{100, 100}, {150, 150}, {200, 200}, {500, 500}, {550, 550},
	}
	for i := range entities {
		ent := builder.NewEntity(&TransformComponent{Position: positions[i]})
		qt.Insert(ent)
		entities[i] = ent
	}

	// Query a circle that contains the first three
	center := Vector{150, 150}
	radius := 100.0
	result := qt.QueryCircle(center, radius)

	if len(result) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(result))
	}
}

// TestClear tests clearing the quadtree.
func TestClear(t *testing.T) {
	_, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](qt.world)

	// Insert some entities
	for i := 0; i < 5; i++ {
		ent := builder.NewEntity(&TransformComponent{Position: Point{X: 100 + float64(i), Y: 100 + float64(i)}})
		qt.Insert(ent)
	}

	qt.Clear()

	if len(qt.root.objects) != 0 {
		t.Errorf("Expected 0 objects after clear, got %d", len(qt.root.objects))
	}
	if qt.root.children[0] != nil {
		t.Error("Children should be nil after clear")
	}

	// Query should return empty
	result := qt.Query(Rectangle{Min: Vector{0, 0}, Max: Vector{testWorldSize, testWorldSize}})
	if len(result) != 0 {
		t.Errorf("Expected 0 results after clear, got %d", len(result))
	}
}

// TestMaxDepth prevents infinite subdivision.
func TestMaxDepth(t *testing.T) {
	world, qt, _ := setupTestQuadtree()
	builder := ecs.NewMap1[TransformComponent](world)

	// Insert many entities in a small area to force deep subdivision
	numEntities := MaxObjectsPerNode * (MaxDepth + 1)
	for i := 0; i < numEntities; i++ {
		ent := builder.NewEntity(&TransformComponent{Position: Point{X: 1, Y: 1}})
		qt.Insert(ent)
	}

	// Traverse to deepest node
	node := qt.root
	depth := 0
	for node.children[0] != nil {
		node = node.children[0] // Assuming all in top-left
		depth++
	}
	if depth > MaxDepth {
		t.Errorf("Exceeded max depth: %d > %d", depth, MaxDepth)
	}
	/* if len(node.objects) > MaxObjectsPerNode {
		// At max depth, it should hold more than max without further subdivision
		// This is expected behavior
	} */
}

// Benchmark constants
var benchSizes = []int{1000, 10000, 100000}

// BenchmarkInsert measures insertion performance.
func BenchmarkInsert(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(formatSize(size), func(b *testing.B) {
			world := ecs.NewWorld(size)
			bounds := Rectangle{Min: Vector{0, 0}, Max: Vector{testWorldSize, testWorldSize}}
			qt := NewQuadtree(&world, bounds)
			builder := ecs.NewMap1[TransformComponent](&world)

			// Pre-create entities with random positions
			entities := make([]ecs.Entity, size)
			for i := 0; i < size; i++ {
				ent := builder.NewEntity(&TransformComponent{Position: Point{
					X: rand.Float64() * testWorldSize,
					Y: rand.Float64() * testWorldSize,
				}})

				entities[i] = ent
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				qt.Clear() // Reset quadtree for each iteration
				b.StartTimer()
				for _, ent := range entities {
					qt.Insert(ent)
				}
			}
		})
	}
}

// BenchmarkQueryRectangle measures rectangular query performance.
func BenchmarkQueryRectangle(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(formatSize(size), func(b *testing.B) {
			world := ecs.NewWorld(size)
			bounds := Rectangle{Min: Vector{0, 0}, Max: Vector{testWorldSize, testWorldSize}}
			qt := NewQuadtree(&world, bounds)
			builder := ecs.NewMap1[TransformComponent](&world)

			// Insert entities with random positions
			for i := 0; i < size; i++ {
				ent := builder.NewEntity(&TransformComponent{Position: Point{
					X: rand.Float64() * testWorldSize,
					Y: rand.Float64() * testWorldSize,
				}})
				qt.Insert(ent)
			}

			// Query a central rectangle covering ~25% of the area
			queryBounds := Rectangle{
				Min: Vector{testWorldSize * 0.25, testWorldSize * 0.25},
				Max: Vector{testWorldSize * 0.75, testWorldSize * 0.75},
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = qt.Query(queryBounds)
			}
		})
	}
}

// BenchmarkQueryCircle measures circular query performance.
func BenchmarkQueryCircle(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(formatSize(size), func(b *testing.B) {
			world := ecs.NewWorld(size)
			bounds := Rectangle{Min: Vector{0, 0}, Max: Vector{testWorldSize, testWorldSize}}
			qt := NewQuadtree(&world, bounds)
			builder := ecs.NewMap1[TransformComponent](&world)

			// Insert entities with random positions
			for i := 0; i < size; i++ {
				ent := builder.NewEntity(&TransformComponent{Position: Point{
					X: rand.Float64() * testWorldSize,
					Y: rand.Float64() * testWorldSize,
				}})
				qt.Insert(ent)
			}

			// Query a central circle with radius covering ~25% area equivalent
			center := Vector{testWorldSize / 2, testWorldSize / 2}
			radius := testWorldSize * 0.25 * math.Sqrt(math.Pi) // Approximate area match

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = qt.QueryCircle(center, radius)
			}
		})
	}
}

// BenchmarkClear measures clearing performance.
func BenchmarkClear(b *testing.B) {
	for _, size := range benchSizes {
		b.Run(formatSize(size), func(b *testing.B) {
			world := ecs.NewWorld(size)
			bounds := Rectangle{Min: Vector{0, 0}, Max: Vector{testWorldSize, testWorldSize}}
			builder := ecs.NewMap1[TransformComponent](&world)

			// Pre-insert entities
			builder.NewBatch(size, &TransformComponent{Position: Point{
				X: rand.Float64() * testWorldSize,
				Y: rand.Float64() * testWorldSize,
			}})

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				qt := NewQuadtree(&world, bounds)
				// Insert all for each clear
				filter := ecs.NewFilter1[TransformComponent](&world)
				query := filter.Query()
				for query.Next() {
					qt.Insert(query.Entity())
				}
				b.StartTimer()
				qt.Clear()
			}
		})
	}
}

// formatSize formats benchmark sizes (e.g., 1000 -> "1K", 1000000 -> "1M").
func formatSize(size int) string {
	if size >= 1000000 {
		return "1M"
	}
	return fmt.Sprintf("%dK", size/1000)
}
