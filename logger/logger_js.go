//go:build js
// +build js

package logger

import (
	"fmt"
	"syscall/js"
)

var (
	instance *Logger
)

// GetLogger returns a singleton instance of Logger for WebAssembly
func GetLogger() *Logger {
	if instance == nil {
		instance = &Logger{
			isDebug: false, // Default to false
		}
	}
	return instance
}

func (l *Logger) log(level string, format string, v ...any) {
	message := fmt.Sprintf("[%s] "+format, append([]any{level}, v...)...)
	consoleLog := js.Global().Get("console").Get("log").Call("bind", js.Global().Get("console"))
	consoleLog.Invoke(message)
}

// Close does nothing in WASM context as there's no file to close
func (l *Logger) Close() error {
	return nil
}
