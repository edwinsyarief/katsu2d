//go:build js && wasm
// +build js,wasm

package katsu2d

import (
	"fmt"
	"syscall/js"

	"github.com/edwinsyarief/assetpacker"
)

func initAssetReader(name string, key []byte) error {
	assetBin, err := loadAssetPackFromWasm(fmt.Sprintf("./%s", name))
	if err != nil {
		return err
	}
	reader, err := assetpacker.NewAssetReaderFromBytes(assetBin, key)
	if err != nil {
		return err
	}
	assets.reader = reader
	return nil
}

func loadAssetPackFromWasm(path string) ([]byte, error) {
	done := make(chan struct{})
	var result []byte
	var err error

	js.Global().Call("fetch", path).Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		response := args[0]
		if !response.Get("ok").Bool() {
			err = fmt.Errorf("HTTP error, status %d", response.Get("status").Int())
			close(done)
			return nil
		}

		response.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			jsArrayBuffer := args[0]
			jsUint8Array := js.Global().Get("Uint8Array").New(jsArrayBuffer)

			result = make([]byte, jsUint8Array.Get("length").Int())
			js.CopyBytesToGo(result, jsUint8Array)
			close(done)
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
			err = fmt.Errorf("failed to convert response to array buffer: %v", args[0].String())
			close(done)
			return nil
		}))
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
		err = fmt.Errorf("fetch error: %v", args[0].String())
		close(done)
		return nil
	}))

	<-done

	if err != nil {
		return nil, err
	}

	return cleanBinaryData(result), nil
}

// cleanBinaryData strips UTF-8 replacement character sequences if present.
func cleanBinaryData(data []byte) []byte {
	clean := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if i+2 < len(data) && data[i] == 239 && data[i+1] == 191 && data[i+2] == 189 {
			i += 3
		} else {
			clean = append(clean, data[i])
			i++
		}
	}
	return clean
}
