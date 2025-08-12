package logger

type Logger struct {
	isDebug bool
}

// Common methods for both environments
func (self *Logger) SetDebugMode(debug bool) {
	self.isDebug = debug
}

func (self *Logger) Error(format string, v ...any) {
	self.log("ERROR", format, v...)
}

func (self *Logger) Info(format string, v ...any) {
	if self.isDebug {
		self.log("INFO", format, v...)
	}
}

func (self *Logger) Debug(format string, v ...any) {
	if self.isDebug {
		self.log("DEBUG", format, v...)
	}
}
