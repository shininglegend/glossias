package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestNewHandler_JSONHasBetterStackFields(t *testing.T) {
	out := &bytes.Buffer{}
	h := NewHandler(out, &Options{Level: slog.LevelDebug, Format: "json"})
	r := slog.NewRecord(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), slog.LevelError, "boom", 0)
	r.AddAttrs(slog.String("path", "/api/x"), slog.Int("status", 500))
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("expected JSON, got %q: %v", out.String(), err)
	}
	if got["dt"] != "2024-01-01T12:00:00Z" {
		t.Errorf("dt = %v", got["dt"])
	}
	if got["level"] != "ERROR" {
		t.Errorf("level = %v", got["level"])
	}
	if got["message"] != "boom" {
		t.Errorf("message = %v", got["message"])
	}
	if got["path"] != "/api/x" {
		t.Errorf("path = %v", got["path"])
	}
	if got["status"] != float64(500) {
		t.Errorf("status = %v", got["status"])
	}
	if _, ok := got["time"]; ok {
		t.Error("time should be remapped to dt")
	}
	if _, ok := got["msg"]; ok {
		t.Error("msg should be remapped to message")
	}
	if strings.Contains(out.String(), "\n\n") || strings.Count(out.String(), "\n") != 1 {
		t.Errorf("want a single JSON line, got %q", out.String())
	}
}

func TestNewHandler_DefaultNonTTYIsJSON(t *testing.T) {
	out := &bytes.Buffer{}
	h := NewHandler(out, &Options{Level: slog.LevelInfo})
	r := slog.NewRecord(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), slog.LevelInfo, "ok", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if !json.Valid(bytes.TrimSpace(out.Bytes())) {
		t.Fatalf("non-TTY default should be JSON, got %q", out.String())
	}
}

func TestNewHandler_PrettyKeepsHumanFormat(t *testing.T) {
	out := &bytes.Buffer{}
	h := NewHandler(out, &Options{Format: "pretty"})
	r := slog.NewRecord(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), slog.LevelInfo, "test message", 0)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "level=INFO") || !strings.Contains(got, `msg="test message"`) {
		t.Errorf("pretty output = %q", got)
	}
}

func TestHTTPRequestLog(t *testing.T) {
	tests := []struct {
		status int
		level  slog.Level
		msg    string
	}{
		{200, slog.LevelInfo, "request completed"},
		{301, slog.LevelInfo, "request completed"},
		{400, slog.LevelWarn, "request client error"},
		{401, slog.LevelWarn, "request client error"},
		{404, slog.LevelWarn, "request client error"},
		{500, slog.LevelError, "request failed"},
		{503, slog.LevelError, "request failed"},
	}
	for _, tt := range tests {
		level, msg := HTTPRequestLog(tt.status)
		if level != tt.level || msg != tt.msg {
			t.Errorf("status %d: got (%v, %q), want (%v, %q)", tt.status, level, msg, tt.level, tt.msg)
		}
	}
}

func TestBetterStackAttrs_LeavesGroupedKeys(t *testing.T) {
	a := slog.String(slog.MessageKey, "inner")
	got := BetterStackAttrs([]string{"req"}, a)
	if got.Key != slog.MessageKey {
		t.Errorf("grouped msg key = %q, want %q", got.Key, slog.MessageKey)
	}
}

func TestResolveFormat(t *testing.T) {
	t.Setenv("LOG_FORMAT", "")
	if got := resolveFormat("", &bytes.Buffer{}); got != "json" {
		t.Errorf("empty + buffer = %q", got)
	}
	t.Setenv("LOG_FORMAT", "pretty")
	if got := resolveFormat("", &bytes.Buffer{}); got != "pretty" {
		t.Errorf("LOG_FORMAT=pretty = %q", got)
	}
	if got := resolveFormat("json", &bytes.Buffer{}); got != "json" {
		t.Errorf("explicit json should win over env, got %q", got)
	}
	if got := resolveFormat("weird", &bytes.Buffer{}); got != "json" {
		t.Errorf("unknown format = %q", got)
	}
}
