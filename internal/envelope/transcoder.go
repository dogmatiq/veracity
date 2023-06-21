package envelope

import (
	"strings"

	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/internal/protobuf/typedproto"
)

// Transcoder re-encodes messages to different media-types on the fly.
type Transcoder struct {
	// MediaTypes is a map of the message's "portable name" to a list of
	// supported media-types, in order of preference.
	MediaTypes map[string][]string

	// Marshaler is the marshaler to use to unmarshal and marshal messages.
	Marshaler marshalkit.Marshaler
}

// Transcode re-encodes the message in env to one of the supported media-types.
func (t *Transcoder) Transcode(env *envelopepb.Envelope) (*envelopepb.Envelope, bool, error) {
	name := env.GetPortableName()
	current := env.GetMediaType()
	candidates := t.MediaTypes[name]

	// If the existing encoding is supported by the consumer use the envelope
	// without any re-encoding.
	for _, candidate := range candidates {
		if strings.EqualFold(current, candidate) {
			return env, true, nil
		}
	}

	packet := marshalkit.Packet{
		MediaType: current,
		Data:      env.GetData(),
	}

	m, err := t.Marshaler.Unmarshal(packet)
	if err != nil {
		return nil, false, err
	}

	// Otherwise, attempt to marshal the message using the client's requested
	// media-types in order of preference.
	packet, ok, err := t.Marshaler.MarshalAs(m, candidates)
	if !ok || err != nil {
		return nil, ok, err
	}

	env = typedproto.Clone(env)
	env.MediaType = packet.MediaType
	env.Data = packet.Data

	return env, true, nil
}
