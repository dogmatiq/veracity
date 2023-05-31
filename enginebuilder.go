package veracity

import (
	"context"

	"github.com/dogmatiq/configkit"
	"github.com/dogmatiq/veracity/internal/integration"
)

// engineBuilder is a [configkit.RichVisitor] that configures an engine to host
// a Dogma application.
type engineBuilder struct {
	engine *Engine
}

func (b *engineBuilder) VisitRichApplication(ctx context.Context, cfg configkit.RichApplication) error {
	return cfg.RichHandlers().AcceptRichVisitor(ctx, b)
}

func (b *engineBuilder) VisitRichAggregate(ctx context.Context, cfg configkit.RichAggregate) error {
	return nil
}

func (b *engineBuilder) VisitRichProcess(ctx context.Context, cfg configkit.RichProcess) error {
	return nil
}

func (b *engineBuilder) VisitRichIntegration(ctx context.Context, cfg configkit.RichIntegration) error {
	for t := range cfg.MessageTypes().Consumed {
		b.engine.executors[t.ReflectType()] = &integration.CommandExecutor{
			Handler: cfg.Handler(),
		}
	}

	return nil
}

func (b *engineBuilder) VisitRichProjection(ctx context.Context, cfg configkit.RichProjection) error {
	return nil
}
