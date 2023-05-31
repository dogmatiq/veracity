package veracity

// An EngineOption configures the behavior of an [Engine].
type EngineOption interface {
	applyEngineOption(*Engine)
}

// An EngineRunOption configures the behavior of [Engine.Run].
type EngineRunOption interface {
	applyEngineRunOption(*Engine)
}

// option is a concrete implementation of all of the XXXOption interfaces.
type option struct {
	engineOption    func(*Engine)
	engineRunOption func(*Engine)
}

func (o option) applyEngineOption(e *Engine)    { o.engineOption(e) }
func (o option) applyEngineRunOption(e *Engine) { o.engineRunOption(e) }
