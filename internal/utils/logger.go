// Package utils provides structured logging, cleanup management, AI provenance
// markers, and disclaimer formatting for the notebooklm-mcp-server.
package utils

import (
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

// Level represents a log severity level.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelSuccess
	LevelWarning
	LevelError
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// Logger provides structured logging with colored output to stderr.
type Logger struct {
	level    Level
	useColor bool
	stderr   io.Writer
}

// NewLogger creates a new Logger that writes to stderr.
// Color is enabled by default on non-Windows platforms.
func NewLogger() *Logger {
	return &Logger{
		level:    LevelInfo,
		useColor: runtime.GOOS != "windows",
		stderr:   os.Stderr,
	}
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// SetColor enables or disables colored output.
func (l *Logger) SetColor(enabled bool) {
	l.useColor = enabled
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, keysAndValues ...any) {
	l.log(LevelDebug, "DEBUG", colorCyan, msg, keysAndValues...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, keysAndValues ...any) {
	l.log(LevelInfo, "INFO", colorBlue, msg, keysAndValues...)
}

// Success logs a success message.
func (l *Logger) Success(msg string, keysAndValues ...any) {
	l.log(LevelSuccess, "SUCCESS", colorGreen, msg, keysAndValues...)
}

// Warning logs a warning message.
func (l *Logger) Warning(msg string, keysAndValues ...any) {
	l.log(LevelWarning, "WARNING", colorYellow, msg, keysAndValues...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, keysAndValues ...any) {
	l.log(LevelError, "ERROR", colorRed, msg, keysAndValues...)
}

// StdLogger returns a standard library *log.Logger that writes to this Logger
// at the given level. Useful for integrating with libraries that require
// a *log.Logger.
func (l *Logger) StdLogger(level Level) *log.Logger {
	return log.New(&logWriter{logger: l, level: level}, "", 0)
}

func (l *Logger) log(level Level, label, color, msg string, keysAndValues ...any) {
	if level < l.level {
		return
	}

	var sb strings.Builder
	if l.useColor {
		sb.WriteString(color)
	}
	sb.WriteString("[")
	sb.WriteString(label)
	sb.WriteString("] ")
	sb.WriteString(msg)

	if len(keysAndValues) > 0 {
		sb.WriteString(" ")
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				if k, ok := keysAndValues[i].(string); ok {
					sb.WriteString(k)
					sb.WriteString("=")
					sb.WriteString(formatValue(keysAndValues[i+1]))
					if i+2 < len(keysAndValues) {
						sb.WriteString(" ")
					}
				}
			}
		}
	}

	if l.useColor {
		sb.WriteString(colorReset)
	}
	sb.WriteString("\n")

	// Write to stderr — stdout is reserved for MCP JSON-RPC
	_, _ = l.stderr.Write([]byte(sb.String()))
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case error:
		return val.Error()
	default:
		return sprintSimple(val)
	}
}

func sprintSimple(v any) string {
	switch val := v.(type) {
	case nil:
		return "<nil>"
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return itoa(val)
	case int64:
		return i64toa(val)
	case float64:
		return f64toa(val)
	case string:
		return val
	default:
		return "<value>"
	}
}

// Simple integer-to-string conversion to avoid fmt import.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func i64toa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func f64toa(f float64) string {
	if f == 0 {
		return "0"
	}
	neg := f < 0
	if neg {
		f = -f
	}
	intPart := int64(f)
	fracPart := f - float64(intPart)
	intStr := i64toa(intPart)
	fracRounded := int(fracPart*100 + 0.5)
	if fracRounded == 0 {
		if neg {
			return "-" + intStr
		}
		return intStr
	}
	fracStr := itoa(fracRounded)
	for len(fracStr) < 2 {
		fracStr = "0" + fracStr
	}
	if neg {
		return "-" + intStr + "." + fracStr
	}
	return intStr + "." + fracStr
}

// logWriter implements io.Writer and forwards writes to a Logger.
type logWriter struct {
	logger *Logger
	level  Level
}

func (w *logWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n")
	switch w.level {
	case LevelDebug:
		w.logger.Debug(msg)
	case LevelInfo:
		w.logger.Info(msg)
	case LevelSuccess:
		w.logger.Success(msg)
	case LevelWarning:
		w.logger.Warning(msg)
	case LevelError:
		w.logger.Error(msg)
	}
	return len(p), nil
}
