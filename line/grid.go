package line

import (
	"image/color"

	ebimath "github.com/edwinsyarief/ebi-math"
	"github.com/hajimehoshi/ebiten/v2"
)

// GridLine represents a grid of lines that can be rendered efficiently in a single draw call.
type GridLine struct {
	vertices  []ebiten.Vertex
	indices   []uint16
	whiteDot  *ebiten.Image
	position  ebimath.Vector
	size      int
	rows      int
	cols      int
	thickness float64
	color     color.RGBA
}

// NewGridLine creates a new grid with the specified dimensions and properties.
func NewGridLine(position ebimath.Vector, size, rows, cols int, thick float64) *GridLine {
	g := &GridLine{
		position:  position,
		size:      size,
		rows:      rows,
		cols:      cols,
		thickness: thick,
		color:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
		vertices:  make([]ebiten.Vertex, 0),
		indices:   make([]uint16, 0),
		whiteDot:  ebiten.NewImage(1, 1),
	}
	g.whiteDot.Fill(color.White)
	g.buildMesh()
	return g
}

func (g *GridLine) buildMesh() {
	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	totalWidth := float64(g.cols * g.size)
	totalHeight := float64(g.rows * g.size)

	// Horizontal lines
	for i := 0; i <= g.rows; i++ {
		p1 := ebimath.Vector{X: 0, Y: float64(i * g.size)}
		p2 := ebimath.Vector{X: totalWidth, Y: float64(i * g.size)}
		g.addLine(p1, p2)
	}

	// Vertical lines
	for i := 0; i <= g.cols; i++ {
		p1 := ebimath.Vector{X: float64(i * g.size), Y: 0}
		p2 := ebimath.Vector{X: float64(i * g.size), Y: totalHeight}
		g.addLine(p1, p2)
	}
}

func (g *GridLine) addLine(p1, p2 ebimath.Vector) {
	halfThick := g.thickness / 2.0
	dir := p2.Sub(p1)
	if dir.Length() == 0 {
		return
	}
	dir = dir.Normalize()
	normal := dir.Orthogonal()

	v1 := p1.Add(normal.ScaleF(halfThick)).Add(g.position)
	v2 := p1.Sub(normal.ScaleF(halfThick)).Add(g.position)
	v3 := p2.Add(normal.ScaleF(halfThick)).Add(g.position)
	v4 := p2.Sub(normal.ScaleF(halfThick)).Add(g.position)

	r, gr, b, a := g.color.R, g.color.G, g.color.B, g.color.A
	cr, cg, cb, ca := float32(r)/255, float32(gr)/255, float32(b)/255, float32(a)/255

	idx := uint16(len(g.vertices))

	g.vertices = append(g.vertices,
		ebiten.Vertex{DstX: float32(v1.X), DstY: float32(v1.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v2.X), DstY: float32(v2.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v3.X), DstY: float32(v3.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0},
		ebiten.Vertex{DstX: float32(v4.X), DstY: float32(v4.Y), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca, SrcX: 0, SrcY: 0},
	)
	g.indices = append(g.indices, idx, idx+1, idx+2, idx+1, idx+3, idx+2)
}

// SetColor sets the color of the grid.
func (g *GridLine) SetColor(c color.RGBA) {
	g.color = c
	g.buildMesh()
}

// SetPosition sets the position of the grid.
func (g *GridLine) SetPosition(pos ebimath.Vector) {
	g.position = pos
	g.buildMesh()
}

// Draw renders the grid to the screen.
func (g *GridLine) Draw(screen *ebiten.Image, op *ebiten.DrawTrianglesOptions) {
	if len(g.vertices) == 0 {
		return
	}
	screen.DrawTriangles(g.vertices, g.indices, g.whiteDot, op)
}
