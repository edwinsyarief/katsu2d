package katsu2d

import (
	_ "embed"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed internal_assets/shaders/aa_sampling_soft.kage
var _aaSamplingSoft []byte

//go:embed internal_assets/shaders/aa_sampling_sharp.kage
var _aaSamplingSharp []byte

//go:embed internal_assets/shaders/nearest.kage
var _nearest []byte

//go:embed internal_assets/shaders/hermite.kage
var _hermite []byte

//go:embed internal_assets/shaders/bicubic.kage
var _bicubic []byte

//go:embed internal_assets/shaders/bilinear.kage
var _bilinear []byte

//go:embed internal_assets/shaders/src_hermite.kage
var _srcHermite []byte

//go:embed internal_assets/shaders/src_bicubic.kage
var _srcBicubic []byte

//go:embed internal_assets/shaders/src_bilinear.kage
var _srcBilinear []byte

var (
	shaderSources     [scalingFilterEndSentinel][]byte
	shaders           [scalingFilterEndSentinel]*ebiten.Shader
	shaderOpts        ebiten.DrawTrianglesShaderOptions
	shaderVertices    []ebiten.Vertex
	shaderVertIndices []uint16
)

func init() {
	shaderSources[_Nearest] = _nearest
	shaderSources[_AASamplingSoft] = _aaSamplingSoft
	shaderSources[_AASamplingSharp] = _aaSamplingSharp
	shaderSources[_Hermite] = _hermite
	shaderSources[_Bicubic] = _bicubic
	shaderSources[_Bilinear] = _bilinear
	shaderSources[_SrcHermite] = _srcHermite
	shaderSources[_SrcBicubic] = _srcBicubic
	shaderSources[_SrcBilinear] = _srcBilinear
}

func compileShader(filter scalingFilter) {
	var err error
	shaders[filter], err = ebiten.NewShader(shaderSources[filter])
	if err != nil {
		panic("Failed to compile shader filter: " + err.Error())
	}
	if shaderOpts.Uniforms == nil {
		initShaderProperties()
	}
}

func initShaderProperties() {
	shaderVertices = make([]ebiten.Vertex, 4)
	shaderVertIndices = []uint16{0, 1, 3, 3, 1, 2}
	shaderOpts.Uniforms = make(map[string]interface{}, 2)
	for i := range 4 { // doesn't matter unless I start doing color scaling
		shaderVertices[i].ColorR = 1.0
		shaderVertices[i].ColorG = 1.0
		shaderVertices[i].ColorB = 1.0
		shaderVertices[i].ColorA = 1.0
	}
}
