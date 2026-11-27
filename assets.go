package katsu2d

import (
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/edwinsyarief/assetpacker"
)

type assetManager struct {
	mu     sync.RWMutex
	fs     fs.FS
	reader *assetpacker.AssetReader
}

var assets = &assetManager{}

func (m *assetManager) setReader(reader *assetpacker.AssetReader) {
	m.mu.Lock()
	old := m.reader
	m.reader = reader
	m.mu.Unlock()

	if old != nil {
		if c, ok := any(old).(io.Closer); ok {
			_ = c.Close()
		}
	}
}

func initFS(fsys embed.FS) {
	setFS(fsys)
}

func setFS(fsys fs.FS) {
	assets.mu.Lock()
	assets.fs = fsys
	assets.mu.Unlock()
}

func initAssetReaderFromBytes(data, key []byte) error {
	reader, err := assetpacker.NewAssetReaderFromBytes(data, key)
	if err != nil {
		return err
	}
	assets.setReader(reader)
	return nil
}

func closeAssetReader() error {
	assets.mu.Lock()
	old := assets.reader
	assets.reader = nil
	assets.mu.Unlock()

	if old != nil {
		if c, ok := any(old).(io.Closer); ok {
			return c.Close()
		}
	}
	return nil
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
	assets.mu.RLock()
	defer assets.mu.RUnlock()

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
	assets.mu.RLock()
	fsys := assets.fs
	assets.mu.RUnlock()

	if fsys == nil {
		return nil, errors.New("katsu2d: asset filesystem is not initialized")
	}
	return fs.ReadFile(fsys, name)
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
