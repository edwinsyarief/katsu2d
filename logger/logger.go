package logger

type Logger struct {
	isDebug bool
}

// Common methods for both environments
func (l *Logger) SetDebugMode(debug bool) {
	l.isDebug = debug
}

func (l *Logger) Error(format string, v ...any) {
	l.log("ERROR", format, v...)
}

func (l *Logger) Info(format string, v ...any) {
	if l.isDebug {
		l.log("INFO", format, v...)
	}
}

func (l *Logger) Debug(format string, v ...any) {
	if l.isDebug {
		l.log("DEBUG", format, v...)
	}
}
