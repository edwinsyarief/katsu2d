//go:build !js
// +build !js

package katsu2d

import "github.com/edwinsyarief/assetpacker"

func initAssetReader(path string, key []byte) {
	reader, err := assetpacker.NewAssetReader(path, key)
	if err != nil {
		panic(err)
	}

	assets.reader = reader
}
