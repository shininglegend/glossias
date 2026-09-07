package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// NewHandler returns a slog handler for the configured format.
// Production (non-TTY, or LOG_FORMAT=json) writes one JSON object per line
// with Better Stack field names so Live Tail can classify severity.
func NewHandler(out io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = &Options{}
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	if resolveFormat(opts.Format, out) == "pretty" {
		return New(out, opts)
	}
	return slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level:       opts.Level,
		ReplaceAttr: BetterStackAttrs,
	})
}

// BetterStackAttrs renames slog's time/msg keys to Better Stack's reserved
// fields so the event timestamp and message are indexed instead of treated
// as arbitrary JSON.
func BetterStackAttrs(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		a.Key = "dt"
	case slog.MessageKey:
		a.Key = "message"
	}
	return a
}

// HTTPRequestLog maps an HTTP status to a slog level and message so 5xx
// requests show as errors in Better Stack instead of the same INFO line
// as a successful response.
func HTTPRequestLog(status int) (slog.Level, string) {
	switch {
	case status >= 500:
		return slog.LevelError, "request failed"
	case status >= 400:
		return slog.LevelWarn, "request client error"
	default:
		return slog.LevelInfo, "request completed"
	}
}

func resolveFormat(format string, out io.Writer) string {
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
	}
	switch format {
	case "pretty", "text":
		return "pretty"
	case "json":
		return "json"
	case "":
		if isTerminal(out) {
			return "pretty"
		}
		return "json"
	default:
		return "json"
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}
