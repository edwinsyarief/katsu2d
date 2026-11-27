//go:build js && wasm
// +build js,wasm

package katsu2d

import (
	"fmt"
	"strings"
	"sync"
	"syscall/js"
)

// ResolveAssetPath returns path unchanged on JS/WASM builds.
func ResolveAssetPath(path string) string {
	return path
}

func normalizeWasmAssetURL(name string) string {
	if strings.HasPrefix(name, "./") ||
		strings.HasPrefix(name, "../") ||
		strings.HasPrefix(name, "/") ||
		strings.HasPrefix(name, "http://") ||
		strings.HasPrefix(name, "https://") {
		return name
	}
	return "./" + name
}

func initAssetReader(name string, key []byte) error {
	assetBin, err := loadAssetPackFromWasm(normalizeWasmAssetURL(name))
	if err != nil {
		return err
	}
	return initAssetReaderFromBytes(assetBin, key)
}

func loadAssetPackFromWasm(path string) ([]byte, error) {
	done := make(chan struct{})
	var result []byte
	var err error
	var once sync.Once
	finish := func() {
		once.Do(func() { close(done) })
	}

	onBufThen := js.FuncOf(func(this js.Value, args []js.Value) any {
		jsArrayBuffer := args[0]
		jsUint8Array := js.Global().Get("Uint8Array").New(jsArrayBuffer)

		result = make([]byte, jsUint8Array.Get("length").Int())
		js.CopyBytesToGo(result, jsUint8Array)
		finish()
		return nil
	})
	defer onBufThen.Release()

	onBufCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
		var reason any
		if len(args) > 0 {
			reason = args[0].String()
		}
		err = fmt.Errorf("failed to convert response to array buffer: %v", reason)
		finish()
		return nil
	})
	defer onBufCatch.Release()

	onFetchThen := js.FuncOf(func(this js.Value, args []js.Value) any {
		response := args[0]
		if !response.Get("ok").Bool() {
			err = fmt.Errorf("HTTP error, status %d", response.Get("status").Int())
			finish()
			return nil
		}
		response.Call("arrayBuffer").Call("then", onBufThen).Call("catch", onBufCatch)
		return nil
	})
	defer onFetchThen.Release()

	onFetchCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
		var reason any
		if len(args) > 0 {
			reason = args[0].String()
		}
		err = fmt.Errorf("fetch error: %v", reason)
		finish()
		return nil
	})
	defer onFetchCatch.Release()

	js.Global().Call("fetch", path).Call("then", onFetchThen).Call("catch", onFetchCatch)

	<-done

	if err != nil {
		return nil, err
	}

	return result, nil
}
