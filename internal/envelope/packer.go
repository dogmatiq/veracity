package envelope

import (
	"time"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/google/uuid"
)

// Packer puts messages into parcels.
type Packer struct {
	// Site is the (optional) identity of the site that the source application
	// is running within.
	//
	// The site is used to disambiguate between messages from different
	// installations of the same application.
	Site *envelopespec.Identity

	// Application is the identity of the application that is the source of the
	// messages.
	Application *envelopespec.Identity

	// Marshaler is used to marshal messages into envelopes.
	Marshaler marshalkit.ValueMarshaler

	// GenerateID is a function used to generate new message IDs. If it is nil,
	// a UUID is generated.
	GenerateID func() string

	// Now is a function used to get the current time. If it is nil, time.Now()
	// is used.
	Now func() time.Time
}

// Pack returns an envelope containing the given message.
func (p *Packer) Pack(m dogma.Message, options ...PackOption) *envelopespec.Envelope {
	id := p.generateID()
	env := &envelopespec.Envelope{
		MessageId:         id,
		CorrelationId:     id,
		CausationId:       id,
		SourceSite:        p.Site,
		SourceApplication: p.Application,
		CreatedAt:         marshalkit.MustMarshalEnvelopeTime(p.now()),
		Description:       dogma.DescribeMessage(m),
	}

	marshalkit.MustMarshalMessageIntoEnvelope(p.Marshaler, m, env)

	for _, opt := range options {
		opt(env)
	}

	if err := env.Validate(); err != nil {
		panic(err)
	}

	return env
}

// Unpack returns the message contained within an envelope.
func (p *Packer) Unpack(env *envelopespec.Envelope) (dogma.Message, error) {
	return marshalkit.UnmarshalMessageFromEnvelope(p.Marshaler, env)
}

// PackOption is an option that alters the behavior of a Pack operation.
type PackOption func(*envelopespec.Envelope)

// WithCause sets env as the "cause" of the message being packed.
func WithCause(env *envelopespec.Envelope) PackOption {
	return func(e *envelopespec.Envelope) {
		e.CausationId = env.GetMessageId()
		e.CorrelationId = env.GetCorrelationId()
	}
}

// WithHandler sets h as the identity of the handler that is the source of the
// message.
func WithHandler(h *envelopespec.Identity) PackOption {
	return func(e *envelopespec.Envelope) {
		e.SourceHandler = h
	}
}

// WithInstanceID sets the aggregate or process instance ID that is the
// source of the message.
func WithInstanceID(id string) PackOption {
	return func(e *envelopespec.Envelope) {
		e.SourceInstanceId = id
	}
}

// WithScheduledFor sets the scheduled time of a timeout message.
func WithScheduledFor(t time.Time) PackOption {
	return func(e *envelopespec.Envelope) {
		e.ScheduledFor = marshalkit.MustMarshalEnvelopeTime(t)
	}
}

// now returns the current time.
func (p *Packer) now() time.Time {
	now := p.Now
	if now == nil {
		now = time.Now
	}

	return now()
}

// generateID generates a new message ID.
func (p *Packer) generateID() string {
	if p.GenerateID != nil {
		return p.GenerateID()
	}

	return uuid.NewString()
}