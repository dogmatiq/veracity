package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"log/slog"

	"github.com/dogmatiq/example"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
	"github.com/dogmatiq/veracity"
)

func main() {
	e := veracity.New(
		&example.App{},
		veracity.WithJournalStore(&memoryjournal.Store{}),
		veracity.WithKeyValueStore(&memorykv.Store{}),
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
