package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/marshalkit/codec"
	"github.com/dogmatiq/marshalkit/codec/json"
	"github.com/dogmatiq/marshalkit/codec/protobuf"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/integration"
	"github.com/dogmatiq/veracity/internal/projection"
)

type applicationVisitor struct {
	*Config

	EventCoordinator *EventCoordinator

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
		Application: marshalIdentity(cfg.Identity()),
		Marshaler:   c.marshaler,
	}

	return cfg.RichHandlers().AcceptRichVisitor(ctx, c)
}

func (c applicationVisitor) VisitRichAggregate(context.Context, configkit.RichAggregate) error {
	return nil
}

func (c applicationVisitor) VisitRichProcess(context.Context, configkit.RichProcess) error {
	return nil
}

func (c applicationVisitor) VisitRichIntegration(_ context.Context, cfg configkit.RichIntegration) error {
	sup := &integration.Supervisor{
		Handler:         cfg.Handler(),
		HandlerIdentity: marshalIdentity(cfg.Identity()),
		Journals:        c.Persistence.Journals,
		Packer:          c.packer,
		EventRecorder:   c.EventCoordinator,
	}

	exec := &integration.CommandExecutor{
		ExecuteQueue: &sup.ExecuteQueue,
		Packer:       c.packer,
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
	sup := &projection.Supervisor{
		Handler:       cfg.Handler(),
		Packer:        c.packer,
		EventConsumer: c.EventCoordinator,
		StreamIDs:     []*uuidpb.UUID{c.EventCoordinator.StreamID},
	}

	c.Tasks = append(c.Tasks, sup.Run)

	return nil
}

func marshalIdentity(id configkit.Identity) *identitypb.Identity {
	key, err := uuidpb.FromString(id.Key)
	if err != nil {
		panic(err)
	}

	return identitypb.New(id.Name, key)
}
