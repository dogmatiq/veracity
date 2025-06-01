package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dogmatiq/example"
	"github.com/dogmatiq/persistencekit/driver/memory/memoryjournal"
	"github.com/dogmatiq/persistencekit/driver/memory/memorykv"
	"github.com/dogmatiq/veracity"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/sdk/log"
)

func main() {
	logExporter, err := stdoutlog.New()
	if err != nil {
		panic(err)
	}

	e := veracity.New(
		&example.App{},
		veracity.WithJournalStore(&memoryjournal.BinaryStore{}),
		veracity.WithKeyValueStore(&memorykv.BinaryStore{}),
		veracity.WithLoggerProvider(
			log.NewLoggerProvider(
				log.WithProcessor(
					log.NewSimpleProcessor(logExporter),
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
