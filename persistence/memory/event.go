package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/dogmatiq/interopspec/envelopespec"
	"google.golang.org/protobuf/proto"
)

// AggregateEventStore stores event messages produced by aggregates
type AggregateEventStore struct {
	m         sync.RWMutex
	instances map[instanceKey]aggregateInstance
}

type aggregateInstance struct {
	Begin     uint64
	End       uint64
	Revisions [][][]byte
}

// ReadBounds returns the revisions that are the bounds of the relevant
// historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// When loading the instance, only those events from revisions in the half-open
// range [begin, end) should be applied to the aggregate root.
func (s *AggregateEventStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, end uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	return i.Begin, i.End, nil
}

// ReadEvents loads some historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// It returns an implementation-defined number of events sourced from revisions
// in the half-open range [begin, end).
//
// When begin == end there are no more historical events to read. Otherwise,
// call ReadEvents() again with begin = end to continue reading events.
//
// The behavior is undefined if begin is lower than the begin revision returned
// by ReadBounds(). Implementations should return an error in this case.
func (s *AggregateEventStore) ReadEvents(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (events []*envelopespec.Envelope, end uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if begin < i.Begin {
		return nil, 0, fmt.Errorf("revision %d is archived", begin)
	}

	for begin < i.End {
		index := begin - i.Begin
		envelopes := i.Revisions[index]
		begin++

		if len(envelopes) > 0 {
			events, err := unmarshalEnvelopes(envelopes)
			return events, begin, err
		}
	}

	return nil, begin, nil
}

// WriteEvents writes events that were recorded by an aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// begin sets the first revision for the instance such that in the future only
// events from revisions in the half-open range [begin, end + 1) are applied
// when loading the aggregate root. Any revisions before begin may be archived.
//
// Events within archived revisions are still made available to external event
// consumers, but will no longer be needed for loading aggregate roots.
//
// end must be the current end revision, that is, the revision after the most
// recent revision of the instance. Otherwise, an "optimistic concurrency
// control" error occurs and no changes are persisted.
//
// The events slice may be empty, which allows modifying the begin revision
// without recording any new events.
func (s *AggregateEventStore) WriteEvents(
	ctx context.Context,
	hk, id string,
	begin, end uint64,
	events []*envelopespec.Envelope,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	r := s.instances[k]

	if end != r.End {
		return fmt.Errorf("optimistic concurrency conflict, %d is not the next revision", end)
	}

	envelopes, err := marshalEnvelopes(events)
	if err != nil {
		return err
	}

	r.Revisions = append(r.Revisions, envelopes)

	if begin > r.Begin {
		r.Revisions = r.Revisions[begin-r.Begin:]
		r.Begin = begin
	}

	r.End++

	if s.instances == nil {
		s.instances = map[instanceKey]aggregateInstance{}
	}

	s.instances[k] = r

	return nil
}

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
