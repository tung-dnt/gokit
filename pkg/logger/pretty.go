package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
)

const (
	ansiReset  = "\033[0m"
	ansiGray   = "\033[90m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
)

// PrettyHandler is an slog.Handler that writes colorized human-readable output.
type PrettyHandler struct {
	w     io.Writer
	mu    *sync.Mutex
	level slog.Leveler
	attrs []slog.Attr
	group string
}

// NewPrettyHandler returns a PrettyHandler writing to w.
func NewPrettyHandler(w io.Writer, opts *slog.HandlerOptions) *PrettyHandler {
	level := slog.LevelInfo
	if opts != nil && opts.Level != nil {
		level = opts.Level.Level()
	}
	return &PrettyHandler{w: w, mu: &sync.Mutex{}, level: level}
}

// Enabled reports whether the handler handles records at the given level.
func (h *PrettyHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

// Handle writes a colorized log line to the underlying writer.
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	color := levelColor(r.Level)
	ts := r.Time.Format("2006-01-02 15:04:05")

	h.mu.Lock()
	defer h.mu.Unlock()

	_, err := fmt.Fprintf(h.w, "%s %s%-5s%s %s",
		ts, color, r.Level.String(), ansiReset, r.Message)
	if err != nil {
		return err
	}

	// Pre-stored attrs from WithAttrs.
	for _, a := range h.attrs {
		h.writeAttr(a)
	}

	// Inline attrs from the log call.
	r.Attrs(func(a slog.Attr) bool {
		h.writeAttr(a)
		return true
	})

	_, err = fmt.Fprintln(h.w)
	return err
}

func (h *PrettyHandler) writeAttr(a slog.Attr) {
	key := a.Key
	if h.group != "" {
		key = h.group + "." + key
	}
	fmt.Fprintf(h.w, " %s%s%s=%v", ansiGray, key, ansiReset, a.Value.Any()) //nolint:errcheck // best-effort write
}

// WithAttrs returns a new handler with the given attributes pre-applied.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{
		w:     h.w,
		mu:    h.mu,
		level: h.level,
		attrs: append(h.attrs, attrs...),
		group: h.group,
	}
}

// WithGroup returns a new handler with the given group name prepended to keys.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	g := name
	if h.group != "" {
		g = h.group + "." + name
	}
	return &PrettyHandler{
		w:     h.w,
		mu:    h.mu,
		level: h.level,
		attrs: h.attrs,
		group: g,
	}
}

func levelColor(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return ansiRed
	case l >= slog.LevelWarn:
		return ansiYellow
	case l >= slog.LevelInfo:
		return ansiGreen
	default:
		return ansiGray
	}
}
