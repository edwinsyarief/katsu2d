package katsu2d

type scalingFilter uint8

const (
	// Anti-aliased pixel art point sampling. Good default, reasonably
	// performant, decent balance between sharpness and stability during
	// zooms and small movements.
	_AASamplingSoft scalingFilter = iota

	// Like AASamplingSoft, but slightly sharper and slightly less stable
	// during zooms and small movements.
	_AASamplingSharp

	// No interpolation. Sharpest and fastest filter, but can lead
	// to distorted geometry. Very unstable, zooming and small movements
	// will be really jumpy and ugly.
	_Nearest

	// Slightly blurrier than AASamplingSoft and more unstable than
	// AASamplingSharp. Still provides fairly decent results at
	// reasonable performance.
	_Hermite

	// The most expensive filter by quite a lot. Slightly less sharp than
	// Hermite, but quite a bit more stable. Might slightly misrepresent
	// some colors throughout high contrast areas.
	_Bicubic

	// Offered mostly for comparison purposes. Slightly blurrier than
	// Hermite, but quite a bit more stable.
	_Bilinear

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	_SrcHermite

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	_SrcBicubic

	// Offered for comparison purposes only. Non high-resolution aware
	// scaling filter, more similar to what naive scaling will look like.
	// This is what Ebitengine will do by default with the FilterLinear
	// filter.
	_SrcBilinear

	scalingFilterEndSentinel
)
