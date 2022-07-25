package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/dogmatiq/veracity/aggregate"
)

// AggregateStore stores aggregate revisions.
//
// It implements aggregate.RevisionReader and aggregate.RevisionWriter.
type AggregateStore struct {
	m         sync.RWMutex
	instances map[instanceKey]aggregateInstanceX
}

type aggregateInstanceX struct {
	Begin     uint64
	Committed uint64
	End       uint64
	Revisions []aggregateRevision
}

type aggregateRevision struct {
	Begin  uint64
	Events [][]byte
}

// ReadBounds returns the revisions that are the bounds of the relevant
// historical events for a specific aggregate instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// commited is the revision after the most recent committed revision, whereas
// end is the revision after the most recent revision, regardless of whether it
// is committed.
//
// When loading the instance, only those events from revisions in the half-open
// range [begin, committed) should be applied to the aggregate root. committed
// may be less than begin.
func (s *AggregateStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (begin, committed, end uint64, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	return i.Begin, i.Committed, i.End, nil
}

// ReadRevisions loads some historical revisions for a specific aggregate
// instance.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// It returns an implementation-defined number of revisions starting with the
// begin revision.
//
// When the revisions slice is empty there are no more revisions to read.
// Otherwise, begin should be incremented by len(revisions) and ReadEvents()
// called again to continue reading revisions.
//
// The behavior is undefined if begin is lower than the begin revision returned
// by ReadBounds(). Implementations should return an error in this case.
func (s *AggregateStore) ReadRevisions(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (revisions []aggregate.Revision, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if begin < i.Begin /* TODO: only if committed */ {
		return nil, fmt.Errorf("revision %d is archived", begin)
	}

	if begin < i.End {
		index := begin - i.Begin
		rev := i.Revisions[index]

		events, err := unmarshalEnvelopes(rev.Events)
		if err != nil {
			return nil, err
		}

		return []aggregate.Revision{
			{
				Begin:  rev.Begin,
				End:    begin,
				Events: events,
			},
		}, nil
	}

	return nil, nil
}

// PrepareRevision prepares a new revision of an aggregate instance to be
// committed.
//
// hk is the identity key of the aggregate message handler. id is the aggregate
// instance ID.
//
// rev.Begin sets the first revision for the instance. In the future only events
// from revisions in the half-open range [rev.Begin, rev.End + 1) are applied
// when loading the aggregate root. The behavior is undefined if rev.Begin is
// larger than rev.End + 1.
//
// Events from revisions prior to rev.Begin are still made available to external
// event consumers, but will no longer be needed for loading aggregate roots and
// may be archived.
//
// rev.End must be the current end revision, that is, the revision after the
// most recent revision of the instance. Otherwise, an "optimistic concurrency
// control" error occurs and no changes are persisted. The behavior is undefined
// if rev.End is greater than the actual end revision.
func (s *AggregateStore) PrepareRevision(
	ctx context.Context,
	hk, id string,
	rev aggregate.Revision,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if rev.End != i.End {
		return fmt.Errorf("optimistic concurrency conflict, %d is not the next revision", rev.End)
	}

	events, err := marshalEnvelopes(rev.Events)
	if err != nil {
		return err
	}

	i.Revisions = append(i.Revisions, aggregateRevision{
		Begin:  rev.Begin,
		Events: events,
	})

	if rev.Begin > i.Begin {
		i.Revisions = i.Revisions[rev.Begin-i.Begin:]
		i.Begin = rev.Begin
	}

	i.End++

	if s.instances == nil {
		s.instances = map[instanceKey]aggregateInstanceX{}
	}

	s.instances[k] = i

	return nil
}

// CommitRevision commits a prepared revision.
//
// Only once a revision is committed can it be considered part of the
// aggregate's history.
//
// rev must exactly match the revision that was prepared. Care must be taken as
// his may or may not be enforced by the implementation.
//
// It returns an error if the revision does not exist or has already been
// committed.
//
// It returns an error if there are uncommitted revisions before the given
// revision.
func (s *AggregateStore) CommitRevision(
	ctx context.Context,
	hk, id string,
	rev aggregate.Revision,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if rev.End >= i.End {
		return fmt.Errorf("revision %d does not exist", rev.End)
	}

	if rev.End < i.Committed {
		return fmt.Errorf("revision %d is already committed", rev.End)
	}

	if rev.End > i.Committed {
		return fmt.Errorf("cannot commit revision %d, the previous revision is not committed", rev.End)
	}

	i.Committed++
	s.instances[k] = i

	return nil
}
