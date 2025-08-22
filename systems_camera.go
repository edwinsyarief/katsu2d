package katsu2d

type BasicCameraSystem struct{}

func (self *BasicCameraSystem) Update(world *World, dt float64) {
	entities := world.Query(CTBasicCamera)
	for _, entity := range entities {
		comp, ok := world.GetComponent(entity, CTBasicCamera)
		if !ok {
			continue
		}
		camera := comp.(*BasicCameraComponent)
		camera.Update(dt)
	}
}

type CameraSystem struct{}

func (self *CameraSystem) Update(world *World, dt float64) {
	entities := world.Query(CTCamera)
	for _, entity := range entities {
		comp, ok := world.GetComponent(entity, CTCamera)
		if !ok {
			continue
		}
		camera := comp.(*CameraComponent)
		camera.Update(dt)
	}
}
