package tlog

import (
	"context"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

// New returns a logger that writes to the test's log.
func New(t testing.TB) *slog.Logger {
	return slog.New(
		&handler{
			T: t,
		},
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
	h.T.Helper()

	buf := &strings.Builder{}
	buf.WriteString(rec.Message)

	buf.WriteString("  ")
	if len(h.groups) == 0 {
		buf.WriteString("---")
	} else {
		buf.WriteString(strings.Join(h.groups, "."))
	}

	for _, attr := range h.attrs {
		buf.WriteString("  ")
		buf.WriteString(attr.String())
	}

	rec.Attrs(func(attr slog.Attr) bool {
		buf.WriteString("  ")
		buf.WriteString(attr.String())
		return true
	})

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
