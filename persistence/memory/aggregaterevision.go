package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/dogmatiq/veracity/aggregate"
)

// AggregateRevisionStore stores aggregate revisions.
//
// It implements aggregate.RevisionReader and aggregate.RevisionWriter.
type AggregateRevisionStore struct {
	m         sync.RWMutex
	instances map[instanceKey]aggregateInstance
}

type aggregateInstance struct {
	Bounds        aggregate.Bounds
	FirstRevision uint64
	Revisions     []aggregateRevision
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
func (s *AggregateRevisionStore) ReadBounds(
	ctx context.Context,
	hk, id string,
) (aggregate.Bounds, error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	return i.Bounds, nil
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
func (s *AggregateRevisionStore) ReadRevisions(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (revisions []aggregate.Revision, _ error) {
	s.m.RLock()
	defer s.m.RUnlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if begin < i.FirstRevision {
		return nil, fmt.Errorf("revision %d is archived", begin)
	}

	if begin < i.Bounds.End {
		rev := i.Revisions[begin-i.FirstRevision]

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
//
// The behavior is undefined if the most recent revision is uncommitted. It is
// the caller's responsibility to commit an uncommitted revision, as indicated
// by the result of a call to RevisionReader.ReadBounds().
func (s *AggregateRevisionStore) PrepareRevision(
	ctx context.Context,
	hk, id string,
	rev aggregate.Revision,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if rev.End != i.Bounds.End {
		return fmt.Errorf("optimistic concurrency conflict, %d is not the next revision", rev.End)
	}

	if i.Bounds.Uncommitted {
		panic("cannot prepare new revision when there is an uncommitted revision")
	}

	events, err := marshalEnvelopes(rev.Events)
	if err != nil {
		return err
	}

	i.Revisions = append(i.Revisions, aggregateRevision{
		Begin:  rev.Begin,
		Events: events,
	})

	i.Bounds.Begin = rev.Begin
	i.Bounds.End++
	i.Bounds.Uncommitted = true

	if s.instances == nil {
		s.instances = map[instanceKey]aggregateInstance{}
	}

	s.instances[k] = i

	return nil
}

// CommitRevision commits a prepared revision.
//
// Committing a revision indicates that the command that produced it will not be
// retried.
//
// It returns an error if the revision does not exist or has already been
// committed.
//
// The behavior is undefined if rev is greater than the most recent revision.
func (s *AggregateRevisionStore) CommitRevision(
	ctx context.Context,
	hk, id string,
	rev uint64,
) error {
	s.m.Lock()
	defer s.m.Unlock()

	k := instanceKey{hk, id}
	i := s.instances[k]

	if rev >= i.Bounds.End {
		return fmt.Errorf("revision %d does not exist", rev)
	}

	last := i.Bounds.End - 1

	if rev == last {
		if i.Bounds.Uncommitted {
			i.Bounds.Uncommitted = false

			r := i.Revisions[rev-i.FirstRevision]
			i.Revisions = i.Revisions[r.Begin-i.FirstRevision:]
			i.FirstRevision = r.Begin

			s.instances[k] = i

			return nil
		}
	}

	return fmt.Errorf("revision %d is already committed", rev)
}
