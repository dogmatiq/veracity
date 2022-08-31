package eventstore

import (
	"context"
	"errors"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
)

// EventStore is a durable, ordered event store.
type EventStore struct {
	Journal journal.Journal[*JournalRecord]

	events  map[string]struct{}
	version uint64
}

// GetByOffset returns the event at the given offset.
func (s *EventStore) GetByOffset(
	ctx context.Context,
	offset uint64,
) (*envelopespec.Envelope, bool, error) {
	rec, ok, err := s.Journal.Read(ctx, offset)
	if !ok || err != nil {
		return nil, false, err
	}

	return rec.GetAppend().GetEnvelope(), true, nil
}

// Append appends an event to the store.
func (s *EventStore) Append(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	if _, ok := s.events[env.GetMessageId()]; ok {
		return nil
	}

	return s.apply(
		ctx,
		&JournalRecord_Append{
			Append: &AppendRecord{
				Envelope: env,
			},
		},
	)
}

func (s *EventStore) applyAppend(rec *AppendRecord) {
	s.events[rec.GetEnvelope().GetMessageId()] = struct{}{}
}

// load reads all entries from the journal and applies them to the store.
func (s *EventStore) load(ctx context.Context) error {
	if s.events != nil {
		return nil
	}

	s.events = map[string]struct{}{}

	for {
		rec, ok, err := s.Journal.Read(ctx, s.version)
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		rec.GetOneOf().(journalRecord).apply(s)
		s.version++
	}

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (s *EventStore) apply(
	ctx context.Context,
	rec journalRecord,
) error {
	ok, err := s.Journal.Write(
		ctx,
		s.version,
		&JournalRecord{
			OneOf: rec,
		},
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("optimistic concurrency conflict")
	}

	rec.apply(s)
	s.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(s *EventStore)
}

func (x *JournalRecord_Append) apply(s *EventStore) { s.applyAppend(x.Append) }
