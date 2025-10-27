package katsu2d

import "github.com/mlange-42/ark/ecs"

// AnimationSystem updates animations.
type AnimationSystem struct {
	filter *ecs.Filter2[AnimationComponent, SpriteComponent]
}

// NewAnimationSystem creates a new AnimationSystem.
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

func (self *AnimationSystem) Initialize(w *ecs.World) {
	self.filter = self.filter.New(w)
}

// Update advances all active animations in the world by the given delta time.
func (self *AnimationSystem) Update(w *ecs.World, dt float64) {
	query := self.filter.Query()
	for query.Next() {
		anim, spr := query.Get()
		if !anim.Active || len(anim.Frames) == 0 {
			continue
		}
		anim.Elapsed += dt
		if anim.Elapsed >= anim.Speed {
			anim.Elapsed -= anim.Speed
			nf := len(anim.Frames)
			switch anim.Mode {
			case AnimOnce:
				if anim.Current+1 >= nf {
					anim.Current = nf - 1
					anim.Active = false
				} else {
					anim.Current++
				}
			case AnimLoop:
				anim.Current++
				anim.Current %= nf
			case AnimBoomerang:
				if nf > 1 {
					if anim.Direction {
						anim.Current++
						if anim.Current >= nf-1 {
							anim.Current = nf - 1
							anim.Direction = false
						}
					} else {
						anim.Current--
						if anim.Current < 0 {
							anim.Current = 0
							anim.Direction = true
						}
					}
				} else {
					anim.Current = 0
					anim.Active = false
				}
			}
			frame := anim.Frames[anim.Current]
			spr.Bound = frame
		}
	}
}
