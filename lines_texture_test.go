package katsu2d

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func assertClose32(t *testing.T, got, want, tol float32, label string) {
	t.Helper()
	if math.Abs(float64(got-want)) > float64(tol) {
		t.Fatalf("%s: got %v, want %v (tol %v)", label, got, want, tol)
	}
}

func TestLineBuilderTextureStretchAndTileUVs(t *testing.T) {
	tex := ebiten.NewImage(32, 16)

	// Stretch mode across two segments of length 40 and 60 (total 100).
	l := NewLine()
	l.SetWidth(10)
	l.SetTexture(tex)
	l.SetTextureMode(LineTextureStretch)
	l.AddPoint(V(0, 0))
	l.AddPoint(V(40, 0))
	l.AddPoint(V(100, 0))
	l.BuildMesh()

	verts, _ := l.GetMesh()
	if len(verts) != 12 {
		t.Fatalf("expected 12 vertices (2 segments, no joints/caps), got %d", len(verts))
	}
	// Segment 1: [0..40] -> uA=0, uB=0.4*32=12.8; vTop=0.5, vBot=15.5
	assertClose32(t, verts[0].SrcX, 0, 1e-4, "seg1 vA1.SrcX")
	assertClose32(t, verts[0].SrcY, 0.5, 1e-4, "seg1 vA1.SrcY")
	assertClose32(t, verts[1].SrcX, 0, 1e-4, "seg1 vA2.SrcX")
	assertClose32(t, verts[1].SrcY, 15.5, 1e-4, "seg1 vA2.SrcY")
	assertClose32(t, verts[2].SrcX, 12.8, 1e-4, "seg1 vB1.SrcX")
	assertClose32(t, verts[2].SrcY, 0.5, 1e-4, "seg1 vB1.SrcY")
	// Segment 2: [40..100] -> uA=12.8, uB=32
	assertClose32(t, verts[6].SrcX, 12.8, 1e-4, "seg2 vA1.SrcX")
	assertClose32(t, verts[8].SrcX, 32.0, 1e-4, "seg2 vB1.SrcX")

	// Tile mode with square 16x16 texture, width=10, length=20 -> 2 tiles -> uB=32.
	squareTex := ebiten.NewImage(16, 16)
	l2 := NewLine()
	l2.SetWidth(10)
	l2.SetTexture(squareTex)
	l2.SetTextureMode(LineTextureTile)
	l2.SetTextureScale(2.0)
	l2.SetTextureOffset(0.25)
	l2.AddPoint(V(0, 0))
	l2.AddPoint(V(20, 0))
	l2.BuildMesh()

	verts2, _ := l2.GetMesh()
	if len(verts2) != 6 {
		t.Fatalf("expected 6 vertices, got %d", len(verts2))
	}
	// offset 0.25 * 16 = 4.0; du = (20 / (10 * 1.0)) * 16 * 2.0 = 64.0 -> uB = 68.0
	assertClose32(t, verts2[0].SrcX, 4.0, 1e-4, "tile vA1.SrcX")
	assertClose32(t, verts2[2].SrcX, 68.0, 1e-4, "tile vB1.SrcX")

	// Scale clamping for <= 0.
	l2.SetTextureScale(0)
	if l2.textureScale != 1e-4 {
		t.Fatalf("expected clamped textureScale 1e-4, got %v", l2.textureScale)
	}
}

func TestLineBuilderJointsAndCapsCoherentUVs(t *testing.T) {
	tex := ebiten.NewImage(32, 16)
	for _, jm := range []LineJointMode{LineJointBevel, LineJointRound, LineJointSharp} {
		for _, cm := range []LineCapMode{LineCapBox, LineCapRound} {
			l := NewLine()
			l.SetWidth(12)
			l.SetTexture(tex)
			l.SetTextureMode(LineTextureTile)
			l.SetTextureOffset(0.5) // ensures uStart = 16 != 0 so (0, 0) never occurs
			l.SetJointMode(jm)
			l.SetCapMode(cm, cm)
			l.AddPoint(V(0, 0))
			l.AddPoint(V(50, 0))
			l.AddPoint(V(50, 50))
			l.BuildMesh()

			verts, _ := l.GetMesh()
			if len(verts) == 0 {
				t.Fatalf("joint=%d cap=%d produced no vertices", jm, cm)
			}
			for i, v := range verts {
				if v.SrcX < 15.99 || v.SrcY < 0.49 || v.SrcY > 15.51 {
					t.Fatalf("joint=%d cap=%d vert[%d] invalid UV (%v, %v)", jm, cm, i, v.SrcX, v.SrcY)
				}
			}
		}
	}
}

func TestLineBuilderOnePixelHeightClampAndSubImageBounds(t *testing.T) {
	b := NewLineBuilder()
	b.points = []Vector{V(0, 0), V(100, 0)}
	b.width = 10
	b.textureMode = LineTextureStretch
	b.texBounds = image.Rect(10, 20, 42, 21) // 32x1 sub-rect
	b.Build()

	if len(b.vertices) != 6 {
		t.Fatalf("expected 6 vertices, got %d", len(b.vertices))
	}
	assertClose32(t, b.vertices[0].SrcX, 10.0, 1e-4, "sub-rect uA")
	assertClose32(t, b.vertices[2].SrcX, 42.0, 1e-4, "sub-rect uB")
	assertClose32(t, b.vertices[0].SrcY, 20.5, 1e-4, "1px height vTop")
	assertClose32(t, b.vertices[1].SrcY, 20.5, 1e-4, "1px height vBot")
}

func TestLineAndRibbonDrawDoNotMutateCallerOptions(t *testing.T) {
	screen := ebiten.NewImage(64, 64)
	tex := ebiten.NewImage(16, 16)

	l := NewLine()
	l.SetTexture(tex)
	l.SetTextureMode(LineTextureTile)
	l.AddPoint(V(0, 16))
	l.AddPoint(V(64, 16))

	op := &ebiten.DrawTrianglesOptions{}
	l.Draw(screen, op)
	if op.Address != ebiten.AddressUnsafe {
		t.Fatalf("Line.Draw mutated caller op.Address to %v", op.Address)
	}

	rt := NewRibbonTrails().
		SetTexture(tex).
		SetTextureMode(LineTextureStretch)
	rt.AddPoint(V(0, 32))
	rt.AddPoint(V(64, 32))
	rt.Update(1.0 / 60.0)
	rt.Draw(screen, op)
	if op.Address != ebiten.AddressUnsafe {
		t.Fatalf("RibbonTrails.Draw mutated caller op.Address to %v", op.Address)
	}
}

func TestRibbonTrailsTextureStretchTileAndJoints(t *testing.T) {
	tex := ebiten.NewImage(32, 16)

	for _, jm := range []LineJointMode{LineJointBevel, LineJointRound, LineJointSharp} {
		rt := NewRibbonTrails().
			SetWidths(10).
			SetJointMode(jm).
			SetTexture(tex).
			SetTextureMode(LineTextureTile).
			SetTextureScale(1.5).
			SetTextureOffset(0.5)
		rt.AddPoint(V(0, 0))
		rt.AddPoint(V(30, 0))
		rt.AddPoint(V(30, 40))
		rt.Update(1.0 / 60.0)

		if len(rt.vertices) == 0 {
			t.Fatalf("joint=%d produced no ribbon vertices", jm)
		}
		for i, v := range rt.vertices {
			if v.SrcX < 15.99 || v.SrcY < 0.49 || v.SrcY > 15.51 {
				t.Fatalf("joint=%d ribbon vert[%d] invalid UV (%v, %v)", jm, i, v.SrcX, v.SrcY)
			}
		}
	}

	// Stretch mode checks exact cumulative arc length mapping.
	rtStretch := NewRibbonTrails().
		SetWidths(10).
		SetJointMode(LineJointBevel).
		SetTexture(tex).
		SetTextureMode(LineTextureStretch)
	rtStretch.AddPoint(V(0, 0))
	rtStretch.AddPoint(V(25, 0))
	rtStretch.AddPoint(V(100, 0))
	rtStretch.Update(1.0 / 60.0)

	if len(rtStretch.vertices) != 12 {
		t.Fatalf("expected 12 vertices for collinear 2-segment ribbon, got %d", len(rtStretch.vertices))
	}
	assertClose32(t, rtStretch.vertices[0].SrcX, 0.0, 1e-4, "ribbon stretch seg1 start")
	assertClose32(t, rtStretch.vertices[2].SrcX, 8.0, 1e-4, "ribbon stretch seg1 end (25/100 * 32)")
	assertClose32(t, rtStretch.vertices[8].SrcX, 32.0, 1e-4, "ribbon stretch seg2 end")
}

func TestLineAndRibbonTexturedZeroAllocPerRebuild(t *testing.T) {
	tex := ebiten.NewImage(32, 16)

	l := NewLine()
	l.SetTexture(tex)
	l.SetTextureMode(LineTextureTile)
	l.SetJointMode(LineJointRound)
	l.SetCapMode(LineCapRound, LineCapRound)
	l.AddPoint(V(0, 0))
	l.AddPoint(V(50, 0))
	l.AddPoint(V(80, 40))
	l.BuildMesh()

	allocsLine := testing.AllocsPerRun(50, func() {
		l.isDirty = true
		l.BuildMesh()
	})
	if allocsLine != 0 {
		t.Fatalf("expected 0 allocs on Line.BuildMesh rebuild, got %v", allocsLine)
	}

	rt := NewRibbonTrails().
		SetWidths(12, 6).
		SetJointMode(LineJointRound).
		SetTexture(tex).
		SetTextureMode(LineTextureStretch)
	rt.AddPoint(V(0, 0))
	rt.AddPoint(V(40, 0))
	rt.AddPoint(V(70, 35))
	rt.Update(0)

	allocsRibbon := testing.AllocsPerRun(50, func() {
		rt.Update(0)
	})
	if allocsRibbon != 0 {
		t.Fatalf("expected 0 allocs on RibbonTrails.Update rebuild, got %v", allocsRibbon)
	}
}
