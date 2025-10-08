package katsu2d

type TweenComponent struct {
	ID       string  // Tween identifier
	Delay    float64 // Delay before the tween starts
	Start    float64 // Starting value of the tween
	End      float64 // Ending value of the tween
	Duration float64 // Duration of the tween in seconds
	Time     float64 // Current elapsed time, including delay
	EaseType EaseType
	Current  float64 // Current value of the tween
	Finished bool    // Indicates if the tween has completed
}
