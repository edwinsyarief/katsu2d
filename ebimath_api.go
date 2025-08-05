package katsu2d

import (
	"katsu2d/components"

	ebimath "github.com/edwinsyarief/ebi-math"
)

type Vector = ebimath.Vector

var (
	V     func(float64, float64) Vector = ebimath.V
	V2    func(float64) Vector          = ebimath.V2
	VInt  func(int, int) Vector         = ebimath.VInt
	V2Int func(int) Vector              = ebimath.V2Int
)

type Rectangle = ebimath.Rectangle

var (
	NewRectangle func(float64, float64, float64, float64) Rectangle = ebimath.NewRectangle
)

func T() *components.Transform {
	return &components.Transform{
		Transform: ebimath.T(),
	}
}

func TPos(x, y float64) *components.Transform {
	t := T()
	t.SetPosition(V(x, y))
	return t
}
