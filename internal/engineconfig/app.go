package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/marshalkit/codec"
	"github.com/dogmatiq/marshalkit/codec/json"
	"github.com/dogmatiq/marshalkit/codec/protobuf"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration"
)

type applicationVisitor struct {
	*Config

	marshaler marshalkit.ValueMarshaler
	packer    *envelope.Packer
}

func (c applicationVisitor) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	var types []reflect.Type
	for t := range cfg.MessageTypes().All() {
		types = append(types, t.ReflectType())
	}

	m, err := codec.NewMarshaler(
		types,
		[]codec.Codec{
			protobuf.DefaultNativeCodec,
			json.DefaultCodec,
		},
	)
	if err != nil {
		return err
	}

	c.marshaler = m

	c.packer = &envelope.Packer{
		Site:        c.SiteID,
		Application: marshalkit.MustMarshalEnvelopeIdentity(cfg.Identity()),
		Marshaler:   c.marshaler,
	}

	return cfg.RichHandlers().AcceptRichVisitor(ctx, c)
}

func (c applicationVisitor) VisitRichAggregate(ctx context.Context, cfg configkit.RichAggregate) error {
	return nil
}

func (c applicationVisitor) VisitRichProcess(ctx context.Context, cfg configkit.RichProcess) error {
	return nil
}

func (c applicationVisitor) VisitRichIntegration(ctx context.Context, cfg configkit.RichIntegration) error {
	ch := make(chan *integration.EnqueueCommandExchange)

	sup := &integration.Supervisor{
		EnqueueCommand: ch,
		Handler:        cfg.Handler(),
		HandlerKey:     cfg.Identity().Key,
		Journals:       c.Persistence.Journals,
		Packer:         c.packer,
	}

	exec := &integration.CommandExecutor{
		EnqueueCommands: ch,
		Packer:          c.packer,
	}

	c.Tasks = append(c.Tasks, sup.Run)

	if c.Application.Executors == nil {
		c.Application.Executors = map[reflect.Type]dogma.CommandExecutor{}
	}

	for t := range cfg.MessageTypes().Consumed {
		c.Application.Executors[t.ReflectType()] = exec
	}

	return nil
}

func (c applicationVisitor) VisitRichProjection(ctx context.Context, cfg configkit.RichProjection) error {
	return nil
}
