package envelope

import (
	"fmt"
	"time"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/enginekit/protobuf/identitypb"
	"github.com/dogmatiq/enginekit/protobuf/uuidpb"
	"github.com/dogmatiq/marshalkit"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A Packer puts messages into envelopes.
type Packer struct {
	// Site is the (optional) identity of the site that the source application
	// is running within.
	//
	// The site is used to disambiguate between messages from different
	// installations of the same application.
	Site *identitypb.Identity

	// Application is the identity of the application that is the source of the
	// messages.
	Application *identitypb.Identity

	// Marshaler is used to marshal messages into envelopes.
	Marshaler marshalkit.ValueMarshaler

	// GenerateID is a function used to generate new message IDs. If it is nil,
	// a UUID is generated.
	GenerateID func() *uuidpb.UUID

	// Now is a function used to get the current time. If it is nil, time.Now()
	// is used.
	Now func() time.Time
}

// Pack returns an envelope containing the given message.
func (p *Packer) Pack(
	m dogma.Message,
	options ...PackOption,
) *envelopepb.Envelope {
	packet := marshalkit.MustMarshal(p.Marshaler, m)
	_, name, err := packet.ParseMediaType()
	if err != nil {
		// CODE COVERAGE: This branch would require the marshaler to violate its
		// own requirements on the format of the media-type.
		panic(err)
	}

	id := p.generateID()

	env := &envelopepb.Envelope{
		MessageId:         id,
		CorrelationId:     id,
		CausationId:       id,
		SourceSite:        p.Site,
		SourceApplication: p.Application,
		CreatedAt:         p.now(),
		Description:       m.MessageDescription(),
		PortableName:      name,
		MediaType:         packet.MediaType,
		Data:              packet.Data,
	}

	for _, opt := range options {
		opt(env)
	}

	if err := env.Validate(); err != nil {
		panic(err)
	}

	return env
}

// Unpack returns the message contained within an envelope.
func (p *Packer) Unpack(env *envelopepb.Envelope) (dogma.Message, error) {
	packet := marshalkit.Packet{
		MediaType: env.MediaType,
		Data:      env.Data,
	}

	m, err := p.Marshaler.Unmarshal(packet)
	if err != nil {
		return nil, err
	}

	if m, ok := m.(dogma.Message); ok {
		return m, nil
	}

	return nil, fmt.Errorf("'%T' is not a dogma message", m)
}

// now returns the current time.
func (p *Packer) now() *timestamppb.Timestamp {
	if p.Now == nil {
		return timestamppb.Now()
	}

	return timestamppb.New(p.Now())
}

// generateID generates a new message ID.
func (p *Packer) generateID() *uuidpb.UUID {
	if p.GenerateID != nil {
		return p.GenerateID()
	}

	return uuidpb.Generate()
}

// PackOption is an option that alters the behavior of a Pack operation.
type PackOption func(*envelopepb.Envelope)

// WithCause sets env as the "cause" of the message being packed.
func WithCause(env *envelopepb.Envelope) PackOption {
	return func(e *envelopepb.Envelope) {
		e.CausationId = env.MessageId
		e.CorrelationId = env.CorrelationId
	}
}

// WithHandler sets h as the identity of the handler that is the source of the
// message.
func WithHandler(h *identitypb.Identity) PackOption {
	return func(e *envelopepb.Envelope) {
		e.SourceHandler = h
	}
}

// WithInstanceID sets the aggregate or process instance ID that is the
// source of the message.
func WithInstanceID(id string) PackOption {
	return func(e *envelopepb.Envelope) {
		e.SourceInstanceId = id
	}
}

// WithCreatedAt sets the creation time of a message.
func WithCreatedAt(t time.Time) PackOption {
	return func(e *envelopepb.Envelope) {
		e.CreatedAt = timestamppb.New(t)
	}
}

// WithScheduledFor sets the scheduled time of a timeout message.
func WithScheduledFor(t time.Time) PackOption {
	return func(e *envelopepb.Envelope) {
		e.ScheduledFor = timestamppb.New(t)
	}
}
