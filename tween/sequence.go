package tween

type Sequence struct {
	tweens  []*Tween
	current int
}

func NewSequence(tweens ...*Tween) *Sequence {
	return &Sequence{
		tweens: tweens,
	}
}

func (self *Sequence) Update(dt float32) (float32, bool) {
	if self.current >= len(self.tweens) {
		return 0, true // No more tweens to process
	}

	tween := self.tweens[self.current]
	value, done := tween.Update(dt)

	if done {
		self.current++ // Move to the next tween
		if self.current >= len(self.tweens) {
			return value, true // Sequence completed
		}
	}

	return value, false // Continue with the current tween
}

func (self *Sequence) Reset() {
	self.current = 0
	for _, tween := range self.tweens {
		tween.time = 0  // Reset each tween's time
		tween.delay = 0 // Reset each tween's delay
	}
}

func (self *Sequence) SetDelay(delay float32) {
	for _, tween := range self.tweens {
		tween.SetDelay(delay) // Set delay for each tween in the sequence
	}
}

func (self *Sequence) SetCallback(callback func()) {
	if self.current < len(self.tweens) {
		self.tweens[self.current].SetCallback(callback) // Set callback for the current tween
	}
}

func (self *Sequence) Set(time float32) float32 {
	if self.current >= len(self.tweens) {
		return 0 // No more tweens to process
	}

	tween := self.tweens[self.current]
	value := tween.Set(time)

	if time >= tween.duration {
		self.current++ // Move to the next tween if the current one is done
	}

	return value
}

func (self *Sequence) IsFinished() bool {
	return self.current >= len(self.tweens)
}

func (self *Sequence) CurrentValue() float32 {
	if self.current >= len(self.tweens) {
		return 0 // No current tween
	}
	return self.tweens[self.current].CurrentValue()
}

func (self *Sequence) Add(tween *Tween) {
	self.tweens = append(self.tweens, tween)
}

func (self *Sequence) Remove(index int) {
	self.tweens = append(self.tweens[:index], self.tweens[index+1:]...)
}
