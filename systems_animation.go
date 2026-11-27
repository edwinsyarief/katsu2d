package katsu2d

import "github.com/edwinsyarief/teishoku"

// AnimationSystem updates sprite frame animations.
type AnimationSystem struct {
	filter      *teishoku.Filter2[AnimationComponent, SpriteComponent]
	initialized bool
}

// NewAnimationSystem creates a new AnimationSystem.
func NewAnimationSystem() *AnimationSystem {
	return &AnimationSystem{}
}

// Initialize prepares the AnimationSystem's ECS filter.
func (s *AnimationSystem) Initialize(w *teishoku.World) {
	if s.initialized {
		return
	}
	s.filter = teishoku.NewFilter2[AnimationComponent, SpriteComponent](w)
	s.initialized = true
}

// Update advances all active animations in the world by the given delta time.
func (s *AnimationSystem) Update(w *teishoku.World, dt float64) {
	s.filter.EachChunk(func(anims []AnimationComponent, sprites []SpriteComponent) {
		for i := range anims {
			anim := &anims[i]
			nf := len(anim.Frames)
			if !anim.Active || nf == 0 || anim.Speed <= 0 {
				continue
			}

			anim.Elapsed += dt
			if anim.Elapsed >= anim.Speed {
				anim.Elapsed -= anim.Speed
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
					if anim.Current >= nf || anim.Current < 0 {
						anim.Current %= nf
						if anim.Current < 0 {
							anim.Current += nf
						}
					}
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

				if anim.Current < 0 {
					anim.Current = 0
				} else if anim.Current >= nf {
					anim.Current = nf - 1
				}
				sprites[i].Bound = anim.Frames[anim.Current]
			}
		}
	})
}
