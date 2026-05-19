package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	tests := []struct {
		name     string
		setLevel Level
		call     func(*Logger)
		expect   bool
	}{
		{"debug at debug level", LevelDebug, func(l *Logger) { l.Debug("test") }, true},
		{"info at debug level", LevelDebug, func(l *Logger) { l.Info("test") }, true},
		{"success at debug level", LevelDebug, func(l *Logger) { l.Success("test") }, true},
		{"warning at debug level", LevelDebug, func(l *Logger) { l.Warning("test") }, true},
		{"error at debug level", LevelDebug, func(l *Logger) { l.Error("test") }, true},
		{"debug at info level", LevelInfo, func(l *Logger) { l.Debug("test") }, false},
		{"info at info level", LevelInfo, func(l *Logger) { l.Info("test") }, true},
		{"debug at error level", LevelError, func(l *Logger) { l.Debug("test") }, false},
		{"error at error level", LevelError, func(l *Logger) { l.Error("test") }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{level: tt.setLevel, useColor: false, stderr: &buf}
			tt.call(logger)
			output := buf.String()
			if tt.expect && output == "" {
				t.Errorf("expected output, got empty")
			}
			if !tt.expect && output != "" {
				t.Errorf("expected no output, got: %s", output)
			}
		})
	}
}

func TestLogger_Labels(t *testing.T) {
	tests := []struct {
		name   string
		call   func(*Logger)
		label  string
	}{
		{"debug label", func(l *Logger) { l.Debug("msg") }, "DEBUG"},
		{"info label", func(l *Logger) { l.Info("msg") }, "INFO"},
		{"success label", func(l *Logger) { l.Success("msg") }, "SUCCESS"},
		{"warning label", func(l *Logger) { l.Warning("msg") }, "WARNING"},
		{"error label", func(l *Logger) { l.Error("msg") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := &Logger{level: LevelDebug, useColor: false, stderr: &buf}
			tt.call(logger)
			if !strings.Contains(buf.String(), "["+tt.label+"]") {
				t.Errorf("expected label [%s] in output: %s", tt.label, buf.String())
			}
		})
	}
}

func TestLogger_KeyValuePairs(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{level: LevelDebug, useColor: false, stderr: &buf}
	logger.Info("test message", "key1", "value1", "key2", 42)

	output := buf.String()
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("expected key1=value1 in output: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("expected key2=42 in output: %s", output)
	}
}

func TestLogger_NoColor(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{level: LevelDebug, useColor: false, stderr: &buf}
	logger.Info("test")

	output := buf.String()
	if strings.Contains(output, "\033[") {
		t.Errorf("expected no ANSI codes in output: %s", output)
	}
}

func TestLogger_WithColor(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{level: LevelDebug, useColor: true, stderr: &buf}
	logger.Info("test")

	output := buf.String()
	if !strings.Contains(output, "\033[") {
		t.Errorf("expected ANSI codes in output: %s", output)
	}
	if !strings.Contains(output, "\033[0m") {
		t.Errorf("expected color reset in output: %s", output)
	}
}

func TestLogger_StdLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := &Logger{level: LevelDebug, useColor: false, stderr: &buf}
	std := logger.StdLogger(LevelError)

	std.Println("standard log message")

	output := buf.String()
	if !strings.Contains(output, "standard log message") {
		t.Errorf("expected message in output: %s", output)
	}
	if !strings.Contains(output, "[ERROR]") {
		t.Errorf("expected ERROR label in output: %s", output)
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"int64", int64(99), "99"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"nil", nil, "<nil>"},
		{"error", &testError{"oops"}, "oops"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.value)
			if got != tt.want {
				t.Errorf("formatValue(%v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
