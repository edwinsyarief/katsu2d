package katsu2d

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/edwinsyarief/assetpacker"
)

func testAssetKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef") // 32 bytes (AES-256)
}

func packTestAssets(t *testing.T, items []assetpacker.Asset, key []byte) string {
	t.Helper()
	baseDir := t.TempDir()
	srcDir := filepath.Join(baseDir, "src")
	for _, item := range items {
		fullPath := filepath.Join(srcDir, filepath.FromSlash(item.Path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		if err := os.WriteFile(fullPath, item.Content, 0o644); err != nil {
			t.Fatalf("WriteFile failed: %v", err)
		}
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(srcDir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origWD) }()

	pakPath := filepath.Join(baseDir, "bundle.pak")
	if err := assetpacker.PackAssets(items, pakPath, key); err != nil {
		t.Fatalf("PackAssets failed: %v", err)
	}
	return pakPath
}

func TestAssetReaderInitAndReadBundledFile(t *testing.T) {
	e := &Engine{}
	defer func() { _ = e.CloseAssetReader() }()
	if err := e.CloseAssetReader(); err != nil {
		t.Fatalf("CloseAssetReader failed: %v", err)
	}

	if _, err := e.ReadBundledFile("config/game.json"); err == nil {
		t.Fatal("expected error reading bundled file when reader is uninitialized")
	}

	key := testAssetKey()
	jsonData := []byte(`{"title":"katsu2d","fps":60}`)
	// Include UTF-8 replacement sequence (0xEF 0xBF 0xBD) to ensure binary payloads stay intact.
	binData := []byte{0x00, 0x01, 0xEF, 0xBF, 0xBD, 0xFE, 0xFF}

	pakPath := packTestAssets(t, []assetpacker.Asset{
		{Path: "config/game.json", Type: "json", Content: jsonData},
		{Path: "data/raw.bin", Type: "bin", Content: binData},
	}, key)

	if err := e.InitAssetReader(pakPath, key); err != nil {
		t.Fatalf("InitAssetReader failed: %v", err)
	}

	gotJSON, err := e.ReadBundledFile("config/game.json")
	if err != nil {
		t.Fatalf("ReadBundledFile(config/game.json) failed: %v", err)
	}
	if !bytes.Equal(gotJSON, jsonData) {
		t.Fatalf("json mismatch: got %q, want %q", gotJSON, jsonData)
	}

	gotBin, err := e.ReadBundledFile("data/raw.bin")
	if err != nil {
		t.Fatalf("ReadBundledFile(data/raw.bin) failed: %v", err)
	}
	if !bytes.Equal(gotBin, binData) {
		t.Fatalf("binary mismatch: got %x, want %x", gotBin, binData)
	}

	if _, err := e.ReadBundledFile("does/not/exist.txt"); err == nil {
		t.Fatal("expected error for missing bundled asset")
	}
}

func TestAssetReaderFromBytes(t *testing.T) {
	e := &Engine{}
	defer func() { _ = e.CloseAssetReader() }()
	key := testAssetKey()
	want := []byte("hello from in-memory bundle")

	pakPath := packTestAssets(t, []assetpacker.Asset{
		{Path: "hello.txt", Type: "txt", Content: want},
	}, key)

	raw, err := os.ReadFile(pakPath)
	if err != nil {
		t.Fatalf("os.ReadFile failed: %v", err)
	}

	if err := e.InitAssetReaderFromBytes(raw, key); err != nil {
		t.Fatalf("InitAssetReaderFromBytes failed: %v", err)
	}

	got, err := e.ReadBundledFile("hello.txt")
	if err != nil {
		t.Fatalf("ReadBundledFile failed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}

	// Failed initialization with wrong key must return an error and keep the active reader intact.
	wrongKey := []byte("fedcba9876543210fedcba9876543210")
	if err := e.InitAssetReaderFromBytes(raw, wrongKey); err == nil {
		t.Fatal("expected error when initializing with wrong key")
	}
	gotAfterFail, err := e.ReadBundledFile("hello.txt")
	if err != nil {
		t.Fatalf("active reader should remain valid after failed re-init: %v", err)
	}
	if !bytes.Equal(gotAfterFail, want) {
		t.Fatalf("got %q, want %q", gotAfterFail, want)
	}
}

func TestAssetReaderReinitAndClose(t *testing.T) {
	e := &Engine{}
	defer func() { _ = e.CloseAssetReader() }()
	key := testAssetKey()

	pak1 := packTestAssets(t, []assetpacker.Asset{
		{Path: "version.txt", Type: "txt", Content: []byte("v1")},
		{Path: "only1.txt", Type: "txt", Content: []byte("one")},
	}, key)
	pak2 := packTestAssets(t, []assetpacker.Asset{
		{Path: "version.txt", Type: "txt", Content: []byte("v2")},
		{Path: "only2.txt", Type: "txt", Content: []byte("two")},
	}, key)

	if err := e.InitAssetReader(pak1, key); err != nil {
		t.Fatalf("first InitAssetReader failed: %v", err)
	}
	got, err := e.ReadBundledFile("version.txt")
	if err != nil || string(got) != "v1" {
		t.Fatalf("expected v1, got %q (err=%v)", got, err)
	}

	// Re-initializing replaces the previous reader.
	if err := e.InitAssetReader(pak2, key); err != nil {
		t.Fatalf("second InitAssetReader failed: %v", err)
	}
	got, err = e.ReadBundledFile("version.txt")
	if err != nil || string(got) != "v2" {
		t.Fatalf("expected v2, got %q (err=%v)", got, err)
	}
	if _, err := e.ReadBundledFile("only1.txt"); err == nil {
		t.Fatal("expected old bundle asset only1.txt to be unavailable after re-init")
	}
	got, err = e.ReadBundledFile("only2.txt")
	if err != nil || string(got) != "two" {
		t.Fatalf("expected two, got %q (err=%v)", got, err)
	}

	// Closing clears the active reader.
	if err := e.CloseAssetReader(); err != nil {
		t.Fatalf("CloseAssetReader failed: %v", err)
	}
	if _, err := e.ReadBundledFile("version.txt"); err == nil {
		t.Fatal("expected error reading after CloseAssetReader")
	}
	// Closing again is a safe no-op.
	if err := e.CloseAssetReader(); err != nil {
		t.Fatalf("second CloseAssetReader failed: %v", err)
	}
}

func TestAssetReaderConcurrentReads(t *testing.T) {
	e := &Engine{}
	defer func() { _ = e.CloseAssetReader() }()
	key := testAssetKey()
	files := map[string][]byte{
		"a.txt": []byte("alpha-payload"),
		"b.txt": []byte("bravo-payload"),
		"c.txt": []byte("charlie-payload"),
	}
	items := make([]assetpacker.Asset, 0, len(files))
	for p, c := range files {
		items = append(items, assetpacker.Asset{Path: p, Type: "txt", Content: c})
	}
	pakPath := packTestAssets(t, items, key)
	if err := e.InitAssetReader(pakPath, key); err != nil {
		t.Fatalf("InitAssetReader failed: %v", err)
	}

	const goroutines = 16
	const iterations = 50
	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*iterations*len(files))

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				for name, want := range files {
					got, err := e.ReadBundledFile(name)
					if err != nil {
						errCh <- err
						return
					}
					if !bytes.Equal(got, want) {
						errCh <- errors.New("payload mismatch during concurrent read")
						return
					}
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent ReadBundledFile failed: %v", err)
	}
}

func TestAssetSetFS(t *testing.T) {
	assets.mu.RLock()
	prevFS := assets.fs
	assets.mu.RUnlock()
	t.Cleanup(func() { setFS(prevFS) })

	e := &Engine{}
	e.SetFS(nil)
	if _, err := openEmbeddedFileErr("hello.txt"); err == nil {
		t.Fatal("expected error when asset filesystem is nil")
	}

	memFS := fstest.MapFS{
		"hello.txt":       &fstest.MapFile{Data: []byte("hello mapfs")},
		"nested/cfg.json": &fstest.MapFile{Data: []byte(`{"ok":true}`)},
	}
	e.SetFS(memFS)

	got, err := openEmbeddedFileErr("hello.txt")
	if err != nil {
		t.Fatalf("openEmbeddedFileErr(hello.txt) failed: %v", err)
	}
	if string(got) != "hello mapfs" {
		t.Fatalf("got %q, want %q", got, "hello mapfs")
	}

	got, err = openEmbeddedFileErr("nested/cfg.json")
	if err != nil {
		t.Fatalf("openEmbeddedFileErr(nested/cfg.json) failed: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("got %q, want %q", got, `{"ok":true}`)
	}

	if _, err := openEmbeddedFileErr("missing.txt"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist for missing file, got %v", err)
	}
}

func TestAssetResolveAssetPath(t *testing.T) {
	if got := ResolveAssetPath(""); got != "" {
		t.Fatalf("empty path: got %q, want empty string", got)
	}

	tmpDir := t.TempDir()
	absExisting := filepath.Join(tmpDir, "existing.pak")
	if err := os.WriteFile(absExisting, []byte("ok"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if got := ResolveAssetPath(absExisting); got != absExisting {
		t.Fatalf("absolute existing path: got %q, want %q", got, absExisting)
	}

	absMissing := filepath.Join(tmpDir, "missing.pak")
	if got := ResolveAssetPath(absMissing); got != absMissing {
		t.Fatalf("absolute missing path: got %q, want %q", got, absMissing)
	}

	t.Chdir(tmpDir)
	if got := ResolveAssetPath("existing.pak"); got != "existing.pak" {
		t.Fatalf("relative existing path: got %q, want %q", got, "existing.pak")
	}
	if got := ResolveAssetPath("no_such_bundle.pak"); got != "no_such_bundle.pak" {
		t.Fatalf("relative missing path: got %q, want %q", got, "no_such_bundle.pak")
	}
}
