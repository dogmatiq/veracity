package memory

import (
	"github.com/dogmatiq/interopspec/envelopespec"
	"google.golang.org/protobuf/proto"
)

// marshalEnvelopes marshals a slice of envelopes to their binary
// representation.
func marshalEnvelopes(envelopes []*envelopespec.Envelope) ([][]byte, error) {
	result := make([][]byte, 0, len(envelopes))

	for _, env := range envelopes {
		data, err := proto.Marshal(env)
		if err != nil {
			return nil, err
		}

		result = append(result, data)
	}

	return result, nil
}

// unmarshalEnvelopes unmarshals a slice of envelopes from their binary
// representation.
func unmarshalEnvelopes(envelopes [][]byte) ([]*envelopespec.Envelope, error) {
	result := make([]*envelopespec.Envelope, 0, len(envelopes))

	for _, data := range envelopes {
		env := &envelopespec.Envelope{}
		if err := proto.Unmarshal(data, env); err != nil {
			return nil, err
		}

		result = append(result, env)
	}

	return result, nil
}
