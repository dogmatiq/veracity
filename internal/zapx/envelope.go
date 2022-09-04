package zapx

import (
	"github.com/dogmatiq/interopspec/envelopespec"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Envelope returns logging fields for a message envelope.
func Envelope(
	key string,
	env *envelopespec.Envelope,
) zap.Field {
	if env == nil {
		panic("envelope must not be nil")
	}

	return zap.Object(key, (*envelopeLogAdaptor)(env))
}

type envelopeLogAdaptor envelopespec.Envelope

func (a *envelopeLogAdaptor) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("message_id", a.MessageId)
	enc.AddString("causation_id", a.CausationId)
	enc.AddString("correlation_id", a.CorrelationId)
	enc.AddString("type", a.PortableName)
	enc.AddString("description", a.Description)
	return nil
}
