package parcel

import (
	"time"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
)

// A Parcel is a container for an envelope and the original information that was
// used to create it.
type Parcel struct {
	// Envelope is the message envelope.
	Envelope *envelopespec.Envelope

	// Message is the original representation of the message.
	Message dogma.Message

	// CreatedAt is the time at which the message was created.
	CreatedAt time.Time

	// ScheduledFor is the time at which a timeout message is scheduled to
	// occur. If the message is not a timeout message it is the zero-value.
	ScheduledFor time.Time
}
