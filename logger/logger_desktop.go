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

type platformLogger struct {
	mu     sync.Mutex
	file   *os.File
	stdLog *log.Logger
}

var (
	instance *Logger
	once     sync.Once
)

// GetLogger returns the singleton Logger instance.
// By default it writes to log.Default() without creating directories or files.
func GetLogger() *Logger {
	once.Do(func() {
		instance = &Logger{
			platformLogger: platformLogger{
				stdLog: log.Default(),
			},
		}
	})
	return instance
}

// EnableFileLogging opens a daily log file inside logDir and directs log output to it.
// Calling EnableFileLogging again closes any previously opened log file first.
func (l *Logger) EnableFileLogging(logDir string) error {
	if logDir == "" {
		logDir = "logs"
	}
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("game_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		_ = l.file.Close()
	}
	l.file = file
	l.stdLog = log.New(file, "", log.LstdFlags)
	return nil
}

func (l *Logger) log(level string, format string, v ...any) {
	msg := fmt.Sprintf("[%s] "+format, append([]any{level}, v...)...)
	l.mu.Lock()
	stdLog := l.stdLog
	if stdLog == nil {
		stdLog = log.Default()
	}
	_ = stdLog.Output(3, msg)
	l.mu.Unlock()
}

// Close closes the open log file if file logging was enabled and resets output to log.Default().
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		l.stdLog = log.Default()
		return err
	}
	return nil
}
