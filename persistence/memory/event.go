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
	revisions map[instanceKey]instanceRevisions
}

type instanceRevisions struct {
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
// begin is the earliest (inclusive) revision that is considered relevant to the
// current state of the instance. It begins as zero, but is advanced as events
// are archived.
//
// end is latest (exclusive) revision of the instance. Because it is exclusive
// it is equivalent to the next revision of this instance.
func (s *AggregateEventStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, end uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	r := s.revisions[k]

	return r.Begin, r.End, nil
}

// ReadEvents loads some historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// begin is the earliest (inclusive) revision to include in the result.
//
// events is the set of events starting at the begin revision.
//
// If begin >= end there are no subsequent historical events to be loaded.
// Otherwise, ReadEvents() should be called again with begin set to end to
// continue reading events.
//
// The number of revisions (and hence events) returned by a single call to
// ReadEvents() is implementation defined.
//
// It may return an error if begin refers to an archived revision.
func (s *AggregateEventStore) ReadEvents(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (events []*envelopespec.Envelope, end uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	r := s.revisions[k]

	if begin < r.Begin {
		return nil, 0, fmt.Errorf("revision %d is archived", begin)
	}

	if begin >= r.End {
		return nil, r.End, nil
	}

	for _, data := range r.Revisions[begin-r.Begin] {
		env := &envelopespec.Envelope{}

		if err := proto.Unmarshal(data, env); err != nil {
			return nil, 0, err
		}

		events = append(events, env)
	}

	return events, begin + 1, nil
}

// WriteEvents writes events that were recorded by an aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// end must be the revision after the most recent revision of the instance;
// otherwise an "optimistic concurrency control" error occurs and no changes are
// persisted.
//
// If archive is true, all historical events, including those written by this
// call are archived. Archived events are typically still made available to
// external event consumers, but will no longer be needed for loading aggregate
// roots.
//
// The events slice may be empty, which allows archiving all existing events
// without adding any new events.
func (s *AggregateEventStore) WriteEvents(
	ctx context.Context,
	hk, id string,
	end uint64,
	events []*envelopespec.Envelope,
	archive bool,
) error {
	var envelopes [][]byte

	if !archive {
		for _, env := range events {
			data, err := proto.Marshal(env)
			if err != nil {
				return err
			}

			envelopes = append(envelopes, data)
		}
	}

	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	r := s.revisions[k]

	if end != r.End {
		return fmt.Errorf("optimistic concurrency conflict, %d is not the next revision", end)
	}

	r.End++

	if archive {
		r.Begin = r.End
		r.Revisions = nil
	} else {
		r.Revisions = append(r.Revisions, envelopes)
	}

	if s.revisions == nil {
		s.revisions = map[instanceKey]instanceRevisions{}
	}

	s.revisions[k] = r

	return nil
}
