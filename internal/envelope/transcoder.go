package envelope

import (
	"strings"

	"github.com/dogmatiq/enginekit/protobuf/envelopepb"
	"github.com/dogmatiq/marshalkit"
	"google.golang.org/protobuf/proto"
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
	candidates := t.MediaTypes[env.PortableName]

	// If the existing encoding is supported by the consumer use the envelope
	// without any re-encoding.
	for _, candidate := range candidates {
		if strings.EqualFold(env.MediaType, candidate) {
			return env, true, nil
		}
	}

	packet := marshalkit.Packet{
		MediaType: env.MediaType,
		Data:      env.Data,
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

	env = proto.Clone(env).(*envelopepb.Envelope)
	env.MediaType = packet.MediaType
	env.Data = packet.Data

	return env, true, nil
}
