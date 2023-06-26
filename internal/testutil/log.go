package testutil

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
	"golang.org/x/exp/slog"
)

// NewLogger returns a logger that writes to the test's log.
func NewLogger(t testing.TB) *slog.Logger {
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
	attrs := slices.Clone(h.attrs)

	if len(h.groups) == 0 {
		rec.Attrs(func(attr slog.Attr) bool {
			attrs = append(attrs, attr)
			return true
		})
	} else {
		var grouped []any
		rec.Attrs(func(attr slog.Attr) bool {
			grouped = append(grouped, attr)
			return true
		})

		var group slog.Attr
		for i := len(h.groups) - 1; i >= 0; i-- {
			group = slog.Group(h.groups[i], grouped...)
			grouped = []any{group}
		}

		attrs = append(attrs, group)
	}

	level := rec.Level.String()
	buf.WriteString(level)

	if len(attrs) == 0 {
		buf.WriteString(" ─ ")
	} else {
		buf.WriteString(" ┬ ")
	}
	buf.WriteString(rec.Message)

	const testLogPrefixWidth = 8
	indent := len(level) + testLogPrefixWidth
	writeAttrs(buf, indent, 0, attrs, true)

	h.T.Log(buf.String())

	return nil
}

func writeAttrs(
	buf *strings.Builder,
	indent, depth int,
	attrs []slog.Attr,
	lastInParent bool,
) {
	if len(attrs) == 0 {
		return
	}

	width := 0
	for _, attr := range attrs {
		if len(attr.Key) > width {
			width = len(attr.Key)
		}
	}

	seen := map[string]struct{}{}

	for i, attr := range attrs {
		if _, ok := seen[attr.Key]; ok {
			continue
		}
		seen[attr.Key] = struct{}{}

		last := i == len(attrs)-1

		buf.WriteByte('\n')

		for i := 0; i < indent; i++ {
			buf.WriteByte(' ')
		}

		for i := 0; i < depth; i++ {
			if lastInParent {
				buf.WriteString("  ")
			} else {
				buf.WriteString("│ ")
			}
		}

		if last {
			buf.WriteString("╰─")
		} else {
			buf.WriteString("├─")
		}

		if attr.Value.Kind() == slog.KindGroup {
			buf.WriteString("┬ ")
			buf.WriteString(attr.Key)
			writeAttrs(buf, indent, depth+1, attr.Value.Group(), last)
		} else {
			buf.WriteString("─ ")
			buf.WriteString(attr.Key)
			buf.WriteString(" ")
			for i := len(attr.Key); i < width+1; i++ {
				buf.WriteString("┈")
			}
			buf.WriteString(" ")

			v := attr.Value.String()
			if strings.ContainsAny(v, " \t\n\r") {
				fmt.Fprintf(buf, "%q", v)
			} else {
				buf.WriteString(v)
			}
		}
	}
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
