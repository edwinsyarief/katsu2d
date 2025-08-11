//go:build js && wasm
// +build js,wasm

package katsu2d

import (
	"fmt"
	"syscall/js"

	"github.com/edwinsyarief/assetpacker"
)

func initAssetReader(name string, key []byte) error {
	// Load the assets.bin from the web environment
	assetBin, err := loadAssetPackFromWasm(fmt.Sprintf("./%s", name))
	if err != nil {
		panic(err)
	}

	// Initialize the asset reader with the binary data instead of a file path
	reader, err := assetpacker.NewAssetReaderFromBytes(assetBin, key)
	if err != nil {
		panic(err)
	}

	assets.reader = reader

	return nil
}

func loadAssetPackFromWasm(path string) ([]byte, error) {
	fmt.Printf("Attempting to load asset from WASM: %s\n", path)

	done := make(chan struct{})
	var result []byte
	var err error

	js.Global().Call("fetch", path).Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		response := args[0]
		// Check if the response is OK
		if !response.Get("ok").Bool() {
			err = fmt.Errorf("HTTP error, status %d", response.Get("status").Int())
			close(done)
			return nil
		}

		// Convert response to ArrayBuffer
		response.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			jsArrayBuffer := args[0]
			jsUint8Array := js.Global().Get("Uint8Array").New(jsArrayBuffer)

			// Convert to Go slice
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

	fmt.Printf("Fetched data length: %d, first 10 bytes: %v, last 10 bytes: %v\n", len(result), result[:10], result[len(result)-10:])

	// Clean binary data if necessary
	cleanedData := cleanBinaryData(result)

	return cleanedData, nil
}

// Clean up the binary data by removing Unicode Replacement Characters
func cleanBinaryData(data []byte) []byte {
	var clean []byte
	for i := 0; i < len(data); {
		if i+2 < len(data) && data[i] == 239 && data[i+1] == 191 && data[i+2] == 189 {
			// Skip the three bytes representing the replacement character
			i += 3
		} else {
			clean = append(clean, data[i])
			i++
		}
	}
	return clean
}
