package veracity

import (
	"context"
	"reflect"
	"sync"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/marshalkit/codec"
	"github.com/dogmatiq/marshalkit/codec/protobuf"
	"github.com/dogmatiq/veracity/persistence"
)

// Engine hosts a Dogma application.
type Engine struct {
	App     dogma.Application
	Journal persistence.Journal

	initOnce  sync.Once
	config    configkit.RichApplication
	marshaler marshalkit.Marshaler
}

// ExecuteCommand enqueues a command for execution.
func (e *Engine) ExecuteCommand(ctx context.Context, m dogma.Message) error {
	e.init()

	// p := e.packer.PackCommand(m)
	// c := &record.Record{
	// 	Op: &record.Record_EnqueueCommand{
	// 		EnqueueCommand: &record.EnqueueCommand{
	// 			Envelope: p.Envelope,
	// 		},
	// 	},
	// }

	// data, err := proto.Marshal(c)
	// if err != nil {
	// 	return err
	// }

	// _, err = e.Journal.Append(ctx, data)
	// return err

	panic("not implemented")
}

// Run starts the engine, processing messages until ctx is canceled or an error
// occurs.
func (e *Engine) Run(ctx context.Context) error {
	e.init()
	panic("not implemented")
}

// init initializes the engine's internal state.
func (e *Engine) init() {
	e.initOnce.Do(func() {
		e.config = configkit.FromApplication(e.App)
		e.marshaler = newMarshaler(e.config)
	})
}

// newMarshaler returns the default marshaler to use for the given application
// configuration.
func newMarshaler(cfg configkit.RichApplication) marshalkit.Marshaler {
	var types []reflect.Type

	for t := range cfg.MessageTypes().All() {
		types = append(types, t.ReflectType())
	}

	cfg.RichHandlers().RangeAggregates(
		func(h configkit.RichAggregate) bool {
			r := h.Handler().New()
			types = append(types, reflect.TypeOf(r))
			return true
		},
	)

	cfg.RichHandlers().RangeProcesses(
		func(h configkit.RichProcess) bool {
			r := h.Handler().New()
			types = append(types, reflect.TypeOf(r))
			return true
		},
	)

	m, err := codec.NewMarshaler(
		types,
		[]codec.Codec{
			protobuf.DefaultNativeCodec,
		},
	)
	if err != nil {
		panic(err)
	}

	return m
}
