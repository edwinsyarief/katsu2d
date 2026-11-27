//go:build !js
// +build !js

package katsu2d

import "github.com/edwinsyarief/assetpacker"

func initAssetReader(path string, key []byte) error {
	reader, err := assetpacker.NewAssetReader(path, key)
	if err != nil {
		return err
	}
	assets.reader = reader
	return nil
}
