package eventstreamtest

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/marshalkit"
	"github.com/dogmatiq/veracity/internal/eventstream"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

// ContainsEvents returns an error if s does not contain the given events.
func ContainsEvents(
	ctx context.Context,
	s *eventstream.EventStream,
	m marshalkit.ValueMarshaler,
	events ...dogma.Message,
) error {
	var offset uint64

	if err := s.Range(
		ctx,
		offset,
		func(
			_ context.Context,
			env *envelopespec.Envelope,
		) (bool, error) {
			if len(events) == 0 {
				return false, errors.New("event stream contains more events than expected")
			}

			expect := events[0]
			events = events[1:]

			actual, err := marshalkit.UnmarshalMessageFromEnvelope(m, env)
			if err != nil {
				return false, err
			}

			if reflect.DeepEqual(actual, expect) {
				offset++
				return true, nil
			}

			return false, fmt.Errorf(
				"event at offset %d does not match expectation:\n%s",
				offset,
				cmp.Diff(expect, actual, protocmp.Transform()),
			)
		},
	); err != nil {
		return err
	}

	if len(events) != 0 {
		return errors.New("event stream contains fewer events than expected")
	}

	return nil
}

// ContainsEnvelopes returns an error if s does not contain the given envelopes.
func ContainsEnvelopes(
	ctx context.Context,
	s *eventstream.EventStream,
	envelopes ...*envelopespec.Envelope,
) error {
	var offset uint64

	if err := s.Range(
		ctx,
		offset,
		func(
			_ context.Context,
			actual *envelopespec.Envelope,
		) (bool, error) {
			if len(envelopes) == 0 {
				return false, errors.New("event stream contains more events than expected")
			}

			expect := envelopes[0]
			envelopes = envelopes[1:]

			if proto.Equal(expect, actual) {
				offset++
				return true, nil
			}

			return false, fmt.Errorf(
				"envelope at offset %d does not match expectation:\n%s",
				offset,
				cmp.Diff(expect, actual, protocmp.Transform()),
			)
		},
	); err != nil {
		return err
	}

	if len(envelopes) != 0 {
		return errors.New("event stream contains fewer events than expected")
	}

	return nil
}
