package katsu2d

import (
	"math"
	"sort"
	"time"

	"math/rand/v2"
)

// Rand provides methods for generating random numbers with various distributions.
type Rand struct {
	rnd *rand.Rand
}

// Random creates a new random number generator with the current time as seed.
func Random() *Rand {
	return RandomWidthSeed(time.Now().UnixNano(), time.Now().UnixNano())
}

// RandomWidthSeed initializes a random number generator with a specific seed.
func RandomWidthSeed(seed1, seed2 int64) *Rand {
	return &Rand{
		rnd: rand.New(rand.NewPCG(uint64(seed1), uint64(seed2))),
	}
}

// SetSeed sets the seed for the random number generator, allowing for reproducible randomness.
func (self *Rand) SetSeed(seed int64) {
	self.rnd = rand.New(rand.NewPCG(uint64(seed), uint64(seed)))
}

// Offset generates a random Vector within the given range for both X and Y components.
func (self *Rand) Offset(min, max float64) Vector {
	return Vector{X: self.FloatRange(min, max), Y: self.FloatRange(min, max)}
}

// Chance returns true with the given probability, false otherwise.
func (self *Rand) Chance(probability float64) bool {
	return self.rnd.Float64() <= probability
}

// Bool returns a random boolean value where true has a 50% chance.
func (self *Rand) Bool() bool {
	return self.rnd.Float64() < 0.5
}

// IntRange generates a random integer within the range [min, max].
func (self *Rand) IntRange(min, max int) int {
	return min + self.rnd.IntN(max-min+1)
}

// PositiveInt64 returns a non-negative random int64.
func (self *Rand) PositiveInt64() int64 {
	return self.rnd.Int64()
}

// PositiveInt returns a non-negative random int.
func (self *Rand) PositiveInt() int {
	return self.rnd.Int()
}

// Uint64 returns a random uint64 value.
func (self *Rand) Uint64() uint64 {
	return self.rnd.Uint64()
}

// Float64 returns a random float64 in the range [0.0, 1.0).
func (self *Rand) Float64() float64 {
	return self.rnd.Float64()
}

// NextFloat64 returns a random float64 in the range [0.0, max).
func (self *Rand) NextFloat64(max float64) float64 {
	return self.rnd.Float64() * max
}

// FloatRange returns a random float64 within the specified range [min, max).
func (self *Rand) FloatRange(min, max float64) float64 {
	return min + self.rnd.Float64()*(max-min)
}

// Rad returns a random angle in radians within the range [0, 2π).
func (self *Rand) Rad() float64 {
	return self.FloatRange(0, 2*math.Pi)
}

// VectorRange returns a random Vector within the specified range for both X and Y.
func (self *Rand) VectorRange(min, max Vector) Vector {
	return min.Add(V(self.NextFloat64(max.X-min.X), self.NextFloat64(max.Y-min.Y)))
}

// RandomIndex selects a random index from a slice. Returns -1 if the slice is empty.
func RandomIndex[T any](r *Rand, slice []T) int {
	if len(slice) == 0 {
		return -1
	}
	return r.IntRange(0, len(slice)-1)
}

// RandomElement selects a random element from the slice. Returns the zero value if the slice is empty.
func RandomElement[T any](r *Rand, slice []T) (element T) {
	if len(slice) == 0 {
		return element // Zero value
	}
	if len(slice) == 1 {
		return slice[0]
	}
	return slice[RandomIndex(r, slice)]
}

// RandomChoose selects a random element from the provided elements.
func RandomChoose[T any](r *Rand, elements ...T) (element T) {
	return RandomElement(r, elements)
}

// RandomShuffle shuffles the elements of the slice in place.
func RandomShuffle[T any](r *Rand, slice []T) {
	r.rnd.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

// RandPicker for weighted random selection
// ---------------------------------------
type RandPicker[T any] struct {
	r *Rand

	keys   randPickerKeySlice
	values []T

	threshold float64
	sorted    bool
}

type randPickerKey struct {
	index     int
	threshold float64
}

type randPickerKeySlice []randPickerKey

// Implement sort.Interface for randPickerKeySlice
func (self randPickerKeySlice) Len() int           { return len(self) }
func (self randPickerKeySlice) Less(i, j int) bool { return self[i].threshold < self[j].threshold }
func (self randPickerKeySlice) Swap(i, j int)      { self[i], self[j] = self[j], self[i] }

// RandomPicker creates a new RandPicker with the given random number generator.
func RandomPicker[T any](r *Rand) *RandPicker[T] {
	return &RandPicker[T]{r: r}
}

// Reset clears all options from the picker, resetting it to an empty state.
func (self *RandPicker[T]) Reset() {
	self.keys = self.keys[:0]
	self.values = self.values[:0]
	self.threshold = 0
	self.sorted = false
}

// AddOption adds a new option to the picker with the given weight for selection probability.
func (self *RandPicker[T]) AddOption(value T, weight float64) {
	if weight == 0 {
		return // Zero probability in any case
	}
	self.threshold += weight
	self.keys = append(self.keys, randPickerKey{
		threshold: self.threshold,
		index:     len(self.values),
	})
	self.values = append(self.values, value)
	self.sorted = false
}

// AddOptions adds multiple options to the picker, each with a default weight of 1.
func (self *RandPicker[T]) AddOptions(values ...T) {
	for _, val := range values {
		self.AddOption(val, 1)
	}
}

// IsEmpty checks if there are no options in the picker.
func (self *RandPicker[T]) IsEmpty() bool {
	return len(self.values) == 0
}

// Pick selects a random option based on the weights provided. If no options exist, returns the zero value.
func (self *RandPicker[T]) Pick() T {
	var result T
	if len(self.values) == 0 {
		return result // Zero value
	}
	if len(self.values) == 1 {
		return self.values[0]
	}

	// Sort keys if not already sorted
	if !self.sorted {
		sort.Sort(&self.keys)
		self.sorted = true
	}

	roll := self.r.FloatRange(0, self.threshold)
	i := sort.Search(len(self.keys), func(i int) bool {
		return roll <= self.keys[i].threshold
	})
	if i < len(self.keys) && roll <= self.keys[i].threshold {
		result = self.values[self.keys[i].index]
	} else {
		result = self.values[len(self.values)-1]
	}
	return result
}
