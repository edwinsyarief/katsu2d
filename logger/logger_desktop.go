//go:build !js
// +build !js

package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *Logger
	once     sync.Once
)

// init is called once in the package's init phase
func init() {
	once.Do(func() {
		logger, err := initLogger()
		if err != nil {
			log.Printf("Failed to initialize file logger: %v. Falling back to stdout.", err)
			instance = &Logger{}
		} else {
			instance = logger
		}
	})
}

// GetLogger returns a singleton instance of Logger
func GetLogger() *Logger {
	return instance
}

func initLogger() (*Logger, error) {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("game_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	log.SetOutput(file)
	return &Logger{
		isDebug: false,
	}, nil
}

func (l *Logger) log(level string, format string, v ...any) {
	log.Printf("[%s] "+format, append([]any{level}, v...)...)
}

func (l *Logger) Close() error {
	if l != instance {
		return nil // Only close if this is the singleton instance
	}
	return os.Stdout.Sync() // Sync stdout if used
}
