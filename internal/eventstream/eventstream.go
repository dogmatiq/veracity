package eventstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/protojournal"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
)

// EventStream is a durable, chronologically ordered stream of event messages.
type EventStream struct {
	// Journal is the journal used to store the queue's state.
	Journal journal.Journal

	// Logger is the target for log messages about changes to the event stream.
	Logger *zap.Logger

	version uint64
	offset  uint64
	events  map[string]struct{}
}

// Append adds events to the end of the stream.
func (s *EventStream) Append(
	ctx context.Context,
	envelopes ...*envelopespec.Envelope,
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	r := &JournalRecord_Append{
		Append: &AppendRecord{
			BeginOffset: s.offset,
		},
	}

	for _, env := range envelopes {
		if _, ok := s.events[env.GetMessageId()]; ok {
			s.Logger.Debug(
				"ignored duplicate event",
				zapx.Envelope("event", env),
			)
			continue
		}

		r.Append.Envelopes = append(r.Append.Envelopes, env)
	}

	if len(r.Append.Envelopes) == 0 {
		return nil
	}

	if err := s.append(ctx, r); err != nil {
		return fmt.Errorf("unable to append event(s): %w", err)
	}

	for _, env := range r.Append.Envelopes {
		s.appendEvent(env)

		s.Logger.Debug(
			"event appended to stream",
			zap.Uint64("eventstream_version", s.version),
			zap.Uint64("eventstream_offset", s.offset-1), // the offset of THIS event
			zapx.Envelope("event", env),
		)
	}

	return nil
}

// apply updates the event stream's in-memory state to reflect an append record.
func (x *JournalRecord_Append) apply(s *EventStream) {
	for _, env := range x.Append.Envelopes {
		s.appendEvent(env)
	}
}

// append updates the event stream's in-memory state to include an appended
// event.
func (s *EventStream) appendEvent(env *envelopespec.Envelope) {
	id := env.MessageId
	s.events[id] = struct{}{}
	s.offset++
}

// Range calls fn for each event in the stream beginning at the given offset.
//
// Iteration is stopped if fn returns false or a non-nil error.
func (s *EventStream) Range(
	ctx context.Context,
	offset uint64,
	fn func(context.Context, *envelopespec.Envelope) (bool, error),
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	if offset >= s.offset {
		return nil
	}

	rec, ver, err := s.search(ctx, offset)
	if err != nil {
		return err
	}

	append := rec.GetAppend()
	begin := append.GetBeginOffset()
	envelopes := append.GetEnvelopes()

	for _, env := range envelopes[offset-begin:] {
		ok, err := fn(ctx, env)
		if !ok || err != nil {
			return err
		}
	}

	for {
		ver++

		ok, err := protojournal.Read(ctx, s.Journal, ver, rec)
		if !ok || err != nil {
			return err
		}

		if append := rec.GetAppend(); append != nil {
			for _, env := range append.Envelopes {
				ok, err := fn(ctx, env)
				if !ok || err != nil {
					return err
				}
			}
		}
	}
}

// search performs a binary search to find the record that contains the event at
// the given offset.
//
// It returns the journal that contains the offset and its version within the
// journal.
func (s *EventStream) search(
	ctx context.Context,
	offset uint64,
) (*JournalRecord, uint64, error) {
	rec := &JournalRecord{}

	if offset == 0 {
		_, err := protojournal.Read(ctx, s.Journal, 0, rec)
		return rec, 0, err
	}

	min := uint64(0)
	max := s.version

	for {
		ver := min>>1 + max>>1

		_, err := protojournal.Read(ctx, s.Journal, ver, rec)
		if err != nil {
			return nil, 0, err
		}

		append := rec.GetAppend()

		if offset < append.GetBeginOffset() {
			max = ver
		} else if offset >= append.GetEndOffset() { // TODO: test edge case here
			min = ver + 1
		} else {
			return rec, ver, nil
		}
	}
}

// load reads all records from the journal and applies them to the stream.
func (s *EventStream) load(ctx context.Context) error {
	if s.events != nil {
		return nil
	}

	s.events = map[string]struct{}{}
	rec := &JournalRecord{}

	for {
		ok, err := protojournal.Read(ctx, s.Journal, s.version, rec)
		if err != nil {
			return fmt.Errorf("unable to load event stream: %w", err)
		}
		if !ok {
			break
		}

		rec.GetOneOf().(journalRecord).apply(s)
		s.version++
	}

	s.Logger.Debug(
		"loaded event stream",
		zap.Uint64("eventstream_version", s.version),
		zap.Uint64("eventstream_offset", s.offset),
	)

	return nil
}

// append adds a record to the journal.
func (s *EventStream) append(ctx context.Context, r journalRecord) error {
	ok, err := protojournal.Append(
		ctx,
		s.Journal,
		s.version,
		&JournalRecord{
			OneOf: r,
		},
	)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("optimistic concurrency conflict")
	}

	s.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(s *EventStream)
}
