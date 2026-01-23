// This package provides a slog.Handler that writes logs to an io.Writer.
// Source: https://github.com/golang/example/blob/master/slog-handler-guide/README.md
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// Check implementation
var _ slog.Handler = (*Logger)(nil)

const (
	// ANSI color codes
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorReset  = "\033[0m"
)

type Logger struct {
	opts Options
	goas []groupOrAttrs

	mu  *sync.Mutex
	out io.Writer
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
	// Control color output (ignored if UseJSON is true)
	UseColors bool
	// Use JSON format for structured logging (Betterstack-compatible)
	UseJSON bool
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func New(out io.Writer, opts *Options) *Logger {
	h := &Logger{out: out, mu: &sync.Mutex{}}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *Logger) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *Logger) Handle(ctx context.Context, r slog.Record) error {
	// Use JSON format for production/Betterstack
	if h.opts.UseJSON {
		return h.handleJSON(ctx, r)
	}

	// Original colored format for development
	buf := make([]byte, 0, 1024)

	// Add color prefix based on level
	if h.opts.UseColors {
		buf = append(buf, h.levelColor(r.Level)...)
	}

	// First line: time and level
	firstLineParts := make([]string, 0, 2)
	if !r.Time.IsZero() {
		firstLineParts = append(firstLineParts, fmt.Sprintf("time=%s", r.Time.Format(time.RFC3339)))
	}
	firstLineParts = append(firstLineParts, fmt.Sprintf("level=%s", r.Level.String()))

	// Join first line parts
	for i, part := range firstLineParts {
		if i > 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, part...)
	}

	buf = append(buf, '\n')

	// Second line: source, message, and attributes (indented, with color)
	secondLineParts := make([]string, 0, 10)
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		secondLineParts = append(secondLineParts, fmt.Sprintf("source=%s:%d", f.File, f.Line))
	}
	secondLineParts = append(secondLineParts, fmt.Sprintf("msg=%q", r.Message))

	r.Attrs(func(a slog.Attr) bool {
		secondLineParts = append(secondLineParts, h.formatAttr(a))
		return true
	})

	// Add indented second line
	buf = append(buf, "  "...) // 2 spaces indent
	for i, part := range secondLineParts {
		if i > 0 {
			buf = append(buf, ' ')
		}
		buf = append(buf, part...)
	}

	// Reset color after second line
	if h.opts.UseColors {
		buf = append(buf, colorReset...)
	}
	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

// handleJSON writes logs in JSON format for Betterstack
func (h *Logger) handleJSON(_ context.Context, r slog.Record) error {
	logEntry := make(map[string]interface{})

	// Standard fields for Betterstack
	logEntry["timestamp"] = r.Time.Format(time.RFC3339Nano)
	logEntry["level"] = r.Level.String()
	logEntry["message"] = r.Message

	// Add source location
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		logEntry["source"] = fmt.Sprintf("%s:%d", f.File, f.Line)
	}

	// Add all attributes
	r.Attrs(func(a slog.Attr) bool {
		h.addJSONAttr(logEntry, a)
		return true
	})

	h.mu.Lock()
	defer h.mu.Unlock()

	encoder := json.NewEncoder(h.out)
	return encoder.Encode(logEntry)
}

// addJSONAttr recursively adds attributes to the JSON log entry
func (h *Logger) addJSONAttr(logEntry map[string]interface{}, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}

	switch a.Value.Kind() {
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return
		}
		if a.Key != "" {
			group := make(map[string]interface{})
			for _, ga := range attrs {
				h.addJSONAttr(group, ga)
			}
			logEntry[a.Key] = group
		} else {
			// Inline group attributes
			for _, ga := range attrs {
				h.addJSONAttr(logEntry, ga)
			}
		}
	case slog.KindTime:
		logEntry[a.Key] = a.Value.Time().Format(time.RFC3339Nano)
	default:
		logEntry[a.Key] = a.Value.Any()
	}
}

func (h *Logger) appendAttr(buf []byte, a slog.Attr, indentLevel int) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	// Indent 4 spaces per level.
	buf = fmt.Appendf(buf, "%*s", indentLevel*4, "")
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = fmt.Appendf(buf, "%s: %q\n", a.Key, a.Value.String())
	case slog.KindTime:
		// Write times in a standard way, without the monotonic time.
		buf = fmt.Appendf(buf, "%s: %s\n", a.Key, a.Value.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s:\n", a.Key)
			indentLevel++
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, indentLevel)
		}
	default:
		buf = fmt.Appendf(buf, "%s: %s\n", a.Key, a.Value)
	}
	return buf
}

// formatAttr formats an attribute for single-line output
func (h *Logger) formatAttr(a slog.Attr) string {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return ""
	}

	switch a.Value.Kind() {
	case slog.KindString:
		return fmt.Sprintf("%s=%q", a.Key, a.Value.String())
	case slog.KindTime:
		return fmt.Sprintf("%s=%s", a.Key, a.Value.Time().Format(time.RFC3339))
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return ""
		}
		var parts []string
		for _, ga := range attrs {
			if formatted := h.formatAttr(ga); formatted != "" {
				parts = append(parts, formatted)
			}
		}
		if a.Key != "" {
			return fmt.Sprintf("%s={%s}", a.Key, fmt.Sprintf("%s", parts))
		}
		return fmt.Sprintf("%s", parts)
	default:
		return fmt.Sprintf("%s=%v", a.Key, a.Value)
	}
}

func (h *Logger) withGroupOrAttrs(goa groupOrAttrs) *Logger {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *Logger) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *Logger) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

// Colors
// Modify the Handle method to add color handling
func (h *Logger) levelColor(level slog.Level) string {
	if !h.opts.UseColors {
		return ""
	}

	switch level {
	case slog.LevelError:
		return colorRed
	case slog.LevelWarn:
		return colorYellow
	case slog.LevelInfo:
		return colorGreen
	case slog.LevelDebug:
		return colorBlue
	default:
		return ""
	}
}
