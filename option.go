package veracity

// An EngineOption configures the behavior of an engine.
type EngineOption interface {
	applyEngineOption(*Engine)
}

type option struct {
	engineOption func(*Engine)
}

func (o option) applyEngineOption(e *Engine) {
	if o.engineOption != nil {
		o.engineOption(e)
	}
}
