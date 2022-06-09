package fixtures

import (
	"time"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/marshalkit/fixtures"
	"github.com/dogmatiq/veracity/parcel"
	"github.com/google/uuid"
)

// NewEnvelope returns a new envelope containing the given message.
//
// If id is empty, a new UUID is generated.
//
// times can contain up to two elements, the first is the created time, the
// second is the scheduled-for time.
func NewEnvelope(
	id string,
	m dogma.Message,
	times ...time.Time,
) *envelopespec.Envelope {
	return NewParcel(id, m, times...).Envelope
}

// NewParcel returns a new parcel containing the given message.
//
// If id is empty, a new UUID is generated.
//
// times can contain up to two elements, the first is the created time, the
// second is the scheduled-for time.
func NewParcel(
	id string,
	m dogma.Message,
	times ...time.Time,
) parcel.Parcel {
	if id == "" {
		id = uuid.NewString()
	}

	var createdAt, scheduledFor time.Time

	switch len(times) {
	case 0:
		createdAt = time.Now()
	case 1:
		createdAt = times[0]
	case 2:
		createdAt = times[0]
		scheduledFor = times[1]
	default:
		panic("too many times specified")
	}

	cleanseTime(&createdAt)
	cleanseTime(&scheduledFor)

	p := parcel.Parcel{
		Envelope: &envelopespec.Envelope{
			MessageId:     id,
			CausationId:   "<cause>",
			CorrelationId: "<correlation>",
			SourceApplication: &envelopespec.Identity{
				Name: "<app-name>",
				Key:  "<app-key>",
			},
			SourceHandler: &envelopespec.Identity{
				Name: "<handler-name>",
				Key:  "<handler-key>",
			},
			SourceInstanceId: "<instance>",
			CreatedAt:        marshalkit.MustMarshalEnvelopeTime(createdAt),
			ScheduledFor:     marshalkit.MustMarshalEnvelopeTime(scheduledFor),
			Description:      dogma.DescribeMessage(m),
		},
		Message:      m,
		CreatedAt:    createdAt,
		ScheduledFor: scheduledFor,
	}

	marshalkit.MustMarshalMessageIntoEnvelope(fixtures.Marshaler, m, p.Envelope)

	return p
}

// cleanseTime marshals/unmarshals time to strip any internal state that would
// not be transmitted across the network.
func cleanseTime(t *time.Time) {
	if t.IsZero() {
		*t = time.Time{}
		return
	}

	data, err := t.MarshalText()
	if err != nil {
		panic(err)
	}

	err = t.UnmarshalText(data)
	if err != nil {
		panic(err)
	}
}
