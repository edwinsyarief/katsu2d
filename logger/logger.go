package logger

import "sync/atomic"

// Logger provides leveled logging across desktop and WebAssembly targets.
type Logger struct {
	isDebug atomic.Bool
	platformLogger
}

// SetDebugMode enables or disables INFO and DEBUG log output.
func (l *Logger) SetDebugMode(debug bool) {
	l.isDebug.Store(debug)
}

// Error logs a formatted error message.
func (l *Logger) Error(format string, v ...any) {
	l.log("ERROR", format, v...)
}

// Info logs a formatted informational message when debug mode is enabled.
func (l *Logger) Info(format string, v ...any) {
	if l.isDebug.Load() {
		l.log("INFO", format, v...)
	}
}

// Debug logs a formatted debug message when debug mode is enabled.
func (l *Logger) Debug(format string, v ...any) {
	if l.isDebug.Load() {
		l.log("DEBUG", format, v...)
	}
}
