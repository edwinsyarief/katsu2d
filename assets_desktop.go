//go:build !js
// +build !js

package katsu2d

import (
	"os"
	"path/filepath"

	"github.com/edwinsyarief/assetpacker"
)

// ResolveAssetPath resolves a relative asset path by checking the current working
// directory first, then falling back to the directory containing the executable.
// Empty paths, absolute paths, and paths that do not exist in either location are
// returned unchanged.
func ResolveAssetPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), path)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return path
}

func initAssetReader(path string, key []byte) error {
	reader, err := assetpacker.NewAssetReader(ResolveAssetPath(path), key)
	if err != nil {
		return err
	}
	assets.setReader(reader)
	return nil
}
