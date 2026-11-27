//go:build js
// +build js

package logger

import (
	"fmt"
	"sync"
	"syscall/js"
)

type platformLogger struct{}

var (
	instance *Logger
	once     sync.Once
)

// GetLogger returns a singleton instance of Logger for WebAssembly.
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{}
	})
	return instance
}

// EnableFileLogging is a no-op in WebAssembly contexts where filesystem logging is unavailable.
func (l *Logger) EnableFileLogging(logDir string) error {
	return nil
}

func (l *Logger) log(level string, format string, v ...any) {
	message := fmt.Sprintf("[%s] "+format, append([]any{level}, v...)...)
	js.Global().Get("console").Call("log", message)
}

// Close does nothing in WASM context as there's no file to close.
func (l *Logger) Close() error {
	return nil
}
