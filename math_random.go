package katsu2d

import (
	"math"
	"math/rand/v2"
	"sort"
)

// Rand provides methods for generating random numbers with various distributions.
type Rand struct {
	rnd *rand.Rand
}

// Random creates a new random number generator seeded from the global PRNG.
func Random() *Rand {
	return &Rand{
		rnd: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

// RandomWithSeed initializes a deterministic random number generator with specific seeds.
func RandomWithSeed(seed1, seed2 int64) *Rand {
	return &Rand{
		rnd: rand.New(rand.NewPCG(uint64(seed1), uint64(seed2))),
	}
}

// SetSeed resets the generator with a deterministic seed.
func (r *Rand) SetSeed(seed int64) {
	r.rnd = rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
}

// Offset generates a random Vector with both components in [min, max).
func (r *Rand) Offset(min, max float64) Vector {
	return Vector{X: r.FloatRange(min, max), Y: r.FloatRange(min, max)}
}

// Chance returns true with the given probability in [0, 1].
func (r *Rand) Chance(probability float64) bool {
	return r.rnd.Float64() <= probability
}

// Bool returns a random boolean with a 50% chance of true.
func (r *Rand) Bool() bool {
	return r.rnd.Float64() < 0.5
}

// IntRange generates a random integer in [min, max].
func (r *Rand) IntRange(min, max int) int {
	if max <= min {
		return min
	}
	return min + r.rnd.IntN(max-min+1)
}

// PositiveInt64 returns a non-negative random int64.
func (r *Rand) PositiveInt64() int64 {
	return r.rnd.Int64()
}

// PositiveInt returns a non-negative random int.
func (r *Rand) PositiveInt() int {
	return r.rnd.Int()
}

// Uint64 returns a random uint64 value.
func (r *Rand) Uint64() uint64 {
	return r.rnd.Uint64()
}

// Float64 returns a random float64 in [0.0, 1.0).
func (r *Rand) Float64() float64 {
	return r.rnd.Float64()
}

// NextFloat64 returns a random float64 in [0.0, max).
func (r *Rand) NextFloat64(max float64) float64 {
	return r.rnd.Float64() * max
}

// FloatRange returns a random float64 in [min, max).
func (r *Rand) FloatRange(min, max float64) float64 {
	return min + r.rnd.Float64()*(max-min)
}

// Rad returns a random angle in radians within [0, 2π).
func (r *Rand) Rad() float64 {
	return r.FloatRange(0, 2*math.Pi)
}

// VectorRange returns a random Vector between min and max.
func (r *Rand) VectorRange(min, max Vector) Vector {
	return min.Add(V(r.NextFloat64(max.X-min.X), r.NextFloat64(max.Y-min.Y)))
}

// RandomIndex selects a random index from slice, or -1 if empty.
func RandomIndex[T any](r *Rand, slice []T) int {
	if len(slice) == 0 {
		return -1
	}
	return r.IntRange(0, len(slice)-1)
}

// RandomElement selects a random element from slice, or the zero value if empty.
func RandomElement[T any](r *Rand, slice []T) (element T) {
	if len(slice) == 0 {
		return element
	}
	if len(slice) == 1 {
		return slice[0]
	}
	return slice[RandomIndex(r, slice)]
}

// RandomChoose selects a random element from the provided variadic arguments.
func RandomChoose[T any](r *Rand, elements ...T) (element T) {
	return RandomElement(r, elements)
}

// RandomShuffle shuffles the elements of slice in place.
func RandomShuffle[T any](r *Rand, slice []T) {
	r.rnd.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

// RandPicker performs weighted random selection over a set of values.
type RandPicker[T any] struct {
	r         *Rand
	keys      randPickerKeySlice
	values    []T
	threshold float64
	sorted    bool
}

type randPickerKey struct {
	index     int
	threshold float64
}

type randPickerKeySlice []randPickerKey

func (s randPickerKeySlice) Len() int           { return len(s) }
func (s randPickerKeySlice) Less(i, j int) bool { return s[i].threshold < s[j].threshold }
func (s randPickerKeySlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// RandomPicker creates a new RandPicker backed by r.
func RandomPicker[T any](r *Rand) *RandPicker[T] {
	return &RandPicker[T]{r: r}
}

// Reset clears all options from the picker while preserving capacity.
func (p *RandPicker[T]) Reset() {
	p.keys = p.keys[:0]
	p.values = p.values[:0]
	p.threshold = 0
	p.sorted = false
}

// AddOption registers value with the given positive weight.
func (p *RandPicker[T]) AddOption(value T, weight float64) {
	if weight <= 0 {
		return
	}
	p.threshold += weight
	p.keys = append(p.keys, randPickerKey{
		threshold: p.threshold,
		index:     len(p.values),
	})
	p.values = append(p.values, value)
	p.sorted = false
}

// AddOptions registers multiple options with equal weight 1.
func (p *RandPicker[T]) AddOptions(values ...T) {
	for _, val := range values {
		p.AddOption(val, 1)
	}
}

// IsEmpty reports whether the picker has no options.
func (p *RandPicker[T]) IsEmpty() bool {
	return len(p.values) == 0
}

// Pick selects a weighted random option, or returns the zero value if empty.
func (p *RandPicker[T]) Pick() T {
	var result T
	if len(p.values) == 0 {
		return result
	}
	if len(p.values) == 1 {
		return p.values[0]
	}
	if !p.sorted {
		sort.Sort(&p.keys)
		p.sorted = true
	}
	roll := p.r.FloatRange(0, p.threshold)
	i := sort.Search(len(p.keys), func(i int) bool {
		return roll <= p.keys[i].threshold
	})
	if i < len(p.keys) && roll <= p.keys[i].threshold {
		return p.values[p.keys[i].index]
	}
	return p.values[len(p.values)-1]
}
