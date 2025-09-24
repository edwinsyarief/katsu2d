package katsu2d

import "github.com/edwinsyarief/lazyecs"

type BasicCameraSystem struct{}

func (self *BasicCameraSystem) Update(world *lazyecs.World, dt float64) {
	query := world.Query(CTBasicCamera)
	for query.Next() {
		if cameras, ok := lazyecs.GetComponentSlice[BasicCameraComponent](query); ok {
			for _, c := range cameras {
				c.Update(dt)
			}
		}
	}
}

type CameraSystem struct{}

func (self *CameraSystem) Update(world *lazyecs.World, dt float64) {
	query := world.Query(CTCamera)
	for query.Next() {
		if cameras, ok := lazyecs.GetComponentSlice[CameraComponent](query); ok {
			for _, c := range cameras {
				c.Update(dt)
			}
		}
	}
}
