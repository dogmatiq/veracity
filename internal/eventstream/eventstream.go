package eventstream

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// EventStream is a durable, chronologically ordered stream of events.
type EventStream struct {
	Journal journal.Journal[*JournalRecord]
	Logger  *zap.Logger

	version uint32
	offset  uint32
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
			Offset: s.offset,
		},
	}

	for _, env := range envelopes {
		if _, ok := s.events[env.GetMessageId()]; ok {
			s.Logger.Debug(
				"event ignored because it is already in the stream",
				zap.Object("eventstream", (*logAdaptor)(s)),
				zapx.Envelope("event", env),
			)
		} else {
			r.Append.Envelopes = append(r.Append.Envelopes, env)
		}
	}

	if len(r.Append.Envelopes) == 0 {
		return nil
	}

	if err := s.apply(ctx, r); err != nil {
		return fmt.Errorf("unable to append event(s): %w", err)
	}

	for _, env := range r.Append.Envelopes {
		s.Logger.Debug(
			"event appended to stream",
			zap.Object("eventstream", (*logAdaptor)(s)),
			zapx.Envelope("event", env),
		)
	}

	return nil
}

func (s *EventStream) applyAppend(r *AppendRecord) {
	for _, env := range r.GetEnvelopes() {
		id := env.GetMessageId()
		s.events[id] = struct{}{}
		s.offset++
	}
}

// Range calls fn for each event in the stream beginning at the given offset.
//
// Iteration is stopped if fn returns false or a non-nil error.
func (s *EventStream) Range(
	ctx context.Context,
	offset uint32,
	fn func(context.Context, *envelopespec.Envelope) (bool, error),
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	if offset >= s.offset {
		return nil
	}

	r, v, err := s.search(ctx, offset)
	if err != nil {
		return err
	}

	append := r.GetAppend()
	envelopes := append.GetEnvelopes()

	for _, env := range envelopes[offset-append.GetOffset():] {
		ok, err := fn(ctx, env)
		if !ok || err != nil {
			return err
		}
	}

	for {
		v++

		r, ok, err := s.Journal.Read(ctx, v)
		if !ok || err != nil {
			return err
		}

		if append := r.GetAppend(); append != nil {
			for _, env := range append.GetEnvelopes() {
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
	offset uint32,
) (*JournalRecord, uint32, error) {
	if offset == 0 {
		r, _, err := s.Journal.Read(ctx, 0)
		return r, 0, err
	}

	min := uint32(0)
	max := s.version

	for {
		v := uint32((uint64(min) + uint64(max)) / 2)

		r, _, err := s.Journal.Read(ctx, v)
		if err != nil {
			return nil, 0, err
		}

		append := r.GetAppend()
		begin := append.GetOffset()
		end := begin + uint32(len(append.GetEnvelopes()))

		if offset < begin {
			max = v
		} else if offset > end {
			min = v + 1
		} else {
			return r, v, nil
		}
	}
}

// load reads all entries from the journal and applies them to the stream.
func (s *EventStream) load(ctx context.Context) error {
	if s.events != nil {
		return nil
	}

	s.events = map[string]struct{}{}

	for {
		r, ok, err := s.Journal.Read(ctx, s.version)
		if err != nil {
			return fmt.Errorf("unable to load event stream: %w", err)
		}
		if !ok {
			break
		}

		r.GetOneOf().(journalRecord).apply(s)
		s.version++
	}

	s.Logger.Debug(
		"loaded event stream from journal",
		zap.Object("eventstream", (*logAdaptor)(s)),
	)

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (s *EventStream) apply(
	ctx context.Context,
	r journalRecord,
) error {
	ok, err := s.Journal.Write(
		ctx,
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

	r.apply(s)
	s.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(s *EventStream)
}

func (x *JournalRecord_Append) apply(s *EventStream) { s.applyAppend(x.Append) }

type logAdaptor EventStream

func (a *logAdaptor) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint32("version", a.version)
	enc.AddInt("size", len(a.events))
	return nil
}
