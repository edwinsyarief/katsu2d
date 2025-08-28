package line

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// LineJoiner interface defines the contract for different line joining strategies
type LineJoiner interface {
	BuildMesh(line *Line) ([]ebiten.Vertex, []uint16)
}

// getJoiner returns the appropriate LineJoiner based on the join type
func getJoiner(joinType LineJoinType) LineJoiner {
	switch joinType {
	case LineJoinMiter:
		return &MiterJoiner{}
	case LineJoinBevel:
		return &BevelJoiner{}
	case LineJoinRound:
		return &RoundJoiner{}
	default:
		return &MiterJoiner{}
	}
}
