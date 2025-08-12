package katsu2d

import (
	"embed"
	"encoding/json"
	"os"

	"github.com/edwinsyarief/assetpacker"
)

type assetManager struct {
	fs     embed.FS
	reader *assetpacker.AssetReader
}

var assets = &assetManager{}

func initFS(fs embed.FS) {
	assets.fs = fs
}

func readFile(name string) []byte {
	content, err := os.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return content
}

func openBundledFile(name string) []byte {
	b, err := assets.reader.GetAsset(name)
	if err != nil {
		panic(err)
	}

	return b.Content
}

func openEmbeddedFile(name string) []byte {
	b, err := assets.fs.ReadFile(name)
	if err != nil {
		panic(err)
	}

	return b
}

func readJson(bytes []byte, v any) {
	if err := json.Unmarshal(bytes, v); err != nil {
		panic(err)
	}
}
