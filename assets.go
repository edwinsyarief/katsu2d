package katsu2d

import (
	"embed"
	"encoding/json"
	"errors"
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

func readFileErr(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func readFile(name string) []byte {
	content, err := readFileErr(name)
	if err != nil {
		panic(err)
	}
	return content
}

func openBundledFileErr(name string) ([]byte, error) {
	if assets.reader == nil {
		return nil, errors.New("katsu2d: asset reader is not initialized")
	}
	b, err := assets.reader.GetAsset(name)
	if err != nil {
		return nil, err
	}
	return b.Content, nil
}

func openBundledFile(name string) []byte {
	content, err := openBundledFileErr(name)
	if err != nil {
		panic(err)
	}
	return content
}

func openEmbeddedFileErr(name string) ([]byte, error) {
	return assets.fs.ReadFile(name)
}

func openEmbeddedFile(name string) []byte {
	content, err := openEmbeddedFileErr(name)
	if err != nil {
		panic(err)
	}
	return content
}

func readJson(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
