package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/marshaler"
	"github.com/dogmatiq/enginekit/marshaler/codecs/json"
	"github.com/dogmatiq/enginekit/marshaler/codecs/protobuf"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/veracity/internal/integration"
)

type applicationVisitor struct {
	*Config

	marshaler marshaler.Marshaler
	packer    *envelopepb.Packer
}

func (c applicationVisitor) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	var types []reflect.Type
	for t := range cfg.MessageTypes().All() {
		types = append(types, t.ReflectType())
	}

	m, err := marshaler.New(
		types,
		[]marshaler.Codec{
			protobuf.DefaultNativeCodec,
			json.DefaultCodec,
		},
	)
	if err != nil {
		return err
	}

	c.marshaler = m

	c.packer = &envelopepb.Packer{
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

func (c applicationVisitor) VisitRichProjection(context.Context, configkit.RichProjection) error {
	return nil
}

func marshalIdentity(id configkit.Identity) *identitypb.Identity {
	key, err := uuidpb.Parse(id.Key)
	if err != nil {
		panic(err)
	}

	return identitypb.New(id.Name, key)
}
