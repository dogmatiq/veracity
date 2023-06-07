package engineconfig

import (
	"context"
	"reflect"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/veracity/internal/integration"
)

type applicationVisitor struct {
	*Config
}

func (c applicationVisitor) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	return cfg.RichHandlers().AcceptRichVisitor(ctx, c)
}

func (c applicationVisitor) VisitRichAggregate(ctx context.Context, cfg configkit.RichAggregate) error {
	return nil
}

func (c applicationVisitor) VisitRichProcess(ctx context.Context, cfg configkit.RichProcess) error {
	return nil
}

func (c applicationVisitor) VisitRichIntegration(ctx context.Context, cfg configkit.RichIntegration) error {
	for t := range cfg.MessageTypes().Consumed {
		if c.Application.Executors == nil {
			c.Application.Executors = map[reflect.Type]dogma.CommandExecutor{}
		}

		c.Application.Executors[t.ReflectType()] = &integration.CommandExecutor{
			Handler: cfg.Handler(),
		}
	}

	return nil
}

func (c applicationVisitor) VisitRichProjection(ctx context.Context, cfg configkit.RichProjection) error {
	return nil
}
