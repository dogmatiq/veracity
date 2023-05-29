package tlog

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

// New returns a logger that writes to the test's log.
func New(t testing.TB) *slog.Logger {
	return slog.New(
		&handler{T: t},
	)
}

type handler struct {
	T      testing.TB
	attrs  []slog.Attr
	groups []string
}

func (h *handler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *handler) Handle(_ context.Context, rec slog.Record) error {
	buf := &strings.Builder{}
	buf.WriteString(rec.Message)

	if len(h.attrs) == 0 && rec.NumAttrs() == 0 {
		return nil
	}

	buf.WriteString("  ---")

	for _, attr := range h.attrs {
		buf.WriteString("  ")
		fmt.Fprintf(buf, "%s: %s", attr.Key, attr.Value)
	}

	var attrs []any
	rec.Attrs(func(attr slog.Attr) bool {
		attrs = append(attrs, attr)
		return true
	})

	for i := len(h.groups) - 1; i >= 0; i-- {
		attrs = []any{
			slog.Group(h.groups[i], attrs...),
		}
	}

	for _, attr := range attrs {
		a := attr.(slog.Attr)
		buf.WriteString("  ")
		fmt.Fprintf(buf, "%s: %s", a.Key, a.Value)
	}

	h.T.Log(buf.String())

	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		T:      h.T,
		attrs:  append(slices.Clone(h.attrs), attrs...),
		groups: h.groups,
	}
}

func (h *handler) WithGroup(name string) slog.Handler {
	return &handler{
		T:      h.T,
		attrs:  h.attrs,
		groups: append(slices.Clone(h.groups), name),
	}
}
