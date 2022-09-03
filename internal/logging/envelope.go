package logging

import (
	"github.com/dogmatiq/interopspec/envelopespec"
	"go.uber.org/zap"
)

// EnvelopeFields returns logging fields for a message envelope.
func EnvelopeFields(env *envelopespec.Envelope) []zap.Field {
	if env == nil {
		return nil
	}

	return []zap.Field{
		zap.Namespace("message"),
		zap.String("id", env.GetMessageId()),
		zap.String("type", env.GetPortableName()),
		zap.String("desc", env.GetDescription()),
	}
}
