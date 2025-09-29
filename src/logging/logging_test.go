// Tests written by claude.ai
// logging_test.go
package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogger_New(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantNil bool
	}{
		{
			name:    "nil options",
			opts:    nil,
			wantNil: false,
		},
		{
			name:    "with options",
			opts:    &Options{Level: slog.LevelDebug},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			got := New(out, tt.opts)
			if (got == nil) != tt.wantNil {
				t.Errorf("New() got = %v, want nil = %v", got, tt.wantNil)
			}
		})
	}
}

func TestLogger_Enabled(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  bool
	}{
		{
			name:  "debug level enabled",
			level: slog.LevelDebug,
			want:  true,
		},
		{
			name:  "info level enabled",
			level: slog.LevelInfo,
			want:  true,
		},
		{
			name:  "warn level enabled",
			level: slog.LevelWarn,
			want:  true,
		},
		{
			name:  "error level enabled",
			level: slog.LevelError,
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			opts := &Options{Level: slog.LevelDebug}
			l := New(out, opts)
			if got := l.Enabled(context.Background(), tt.level); got != tt.want {
				t.Errorf("Enabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_Handle(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		record  slog.Record
		want    string
		wantErr bool
	}{
		{
			name: "basic message",
			record: func() slog.Record {
				r := slog.NewRecord(fixedTime, slog.LevelInfo, "test message", 0)
				return r
			}(),
			want: `time=2024-01-01T12:00:00Z level=INFO msg="test message"
`,
		},
		{
			name: "message with attributes",
			record: func() slog.Record {
				r := slog.NewRecord(fixedTime, slog.LevelInfo, "test message", 0)
				r.AddAttrs(slog.String("key", "value"))
				return r
			}(),
			want: `time=2024-01-01T12:00:00Z level=INFO msg="test message" key="value"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			l := New(out, nil)
			err := l.Handle(context.Background(), tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got := out.String(); got != tt.want {
				t.Errorf("Handle()\ngot = %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestLogger_WithGroup(t *testing.T) {
	tests := []struct {
		name      string
		groupName string
		wantSame  bool
	}{
		{
			name:      "empty group name",
			groupName: "",
			wantSame:  true,
		},
		{
			name:      "non-empty group name",
			groupName: "test-group",
			wantSame:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			l := New(out, nil)
			got := l.WithGroup(tt.groupName)
			if (got == l) != tt.wantSame {
				t.Errorf("WithGroup() same instance = %v, want %v", got == l, tt.wantSame)
			}
		})
	}
}

func TestLogger_WithAttrs(t *testing.T) {
	tests := []struct {
		name     string
		attrs    []slog.Attr
		wantSame bool
	}{
		{
			name:     "empty attrs",
			attrs:    nil,
			wantSame: true,
		},
		{
			name:     "non-empty attrs",
			attrs:    []slog.Attr{slog.String("key", "value")},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			l := New(out, nil)
			got := l.WithAttrs(tt.attrs)
			if (got == l) != tt.wantSame {
				t.Errorf("WithAttrs() same instance = %v, want %v", got == l, tt.wantSame)
			}
		})
	}
}

func TestLogger_appendAttr(t *testing.T) {
	tests := []struct {
		name        string
		attr        slog.Attr
		indentLevel int
		want        string
	}{
		{
			name:        "string attribute",
			attr:        slog.String("key", "value"),
			indentLevel: 0,
			want:        `key: "value"` + "\n",
		},
		{
			name:        "time attribute",
			attr:        slog.Time("time", time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)),
			indentLevel: 0,
			want:        "time: 2024-01-01T12:00:00Z\n",
		},
		{
			name:        "empty group",
			attr:        slog.Group("group"),
			indentLevel: 0,
			want:        "",
		},
		{
			name:        "group with content",
			attr:        slog.Group("group", slog.String("key", "value")),
			indentLevel: 0,
			want:        "group:\n    key: \"value\"\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(&bytes.Buffer{}, nil)
			got := l.appendAttr([]byte{}, tt.attr, tt.indentLevel)
			if string(got) != tt.want {
				t.Errorf("appendAttr() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Helper function to test the entire logging flow
func TestLogger_Integration(t *testing.T) {
	out := &bytes.Buffer{}
	logger := New(out, &Options{Level: slog.LevelDebug})

	// Create a structured logger
	slogger := slog.New(logger)

	// Log various types of messages
	slogger.Info("test message",
		"string", "value",
		"number", 42,
		"bool", true)

	output := out.String()

	// Verify expected content
	expectedFields := []string{
		"time=", "level=INFO",
		"msg=\"test message\"",
		"string=\"value\"",
		"number=42",
		"bool=true",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("Output missing expected field: %s", field)
		}
	}
}
