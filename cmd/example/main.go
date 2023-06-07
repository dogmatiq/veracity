package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dogmatiq/example"
	"github.com/dogmatiq/veracity"
	"github.com/dogmatiq/veracity/persistence/driver/memory"
	"golang.org/x/exp/slog"
)

func main() {
	e := veracity.New(
		&example.App{},
		veracity.WithJournalStore(&memory.JournalStore{}),
		veracity.WithKeyValueStore(&memory.KeyValueStore{}),
		veracity.WithLogger(
			slog.New(
				slog.NewJSONHandler(
					os.Stdout,
					&slog.HandlerOptions{
						Level: slog.LevelDebug,
					},
				),
			),
		),
	)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := e.Run(ctx); err != nil {
		panic(err)
	}
}
