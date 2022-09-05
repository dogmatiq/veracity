package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/persistence/journal"
	"github.com/dogmatiq/veracity/internal/zapx"
	"go.uber.org/zap"
)

type InstanceSupervisor struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Journal         journal.Journal[*JournalRecord]
	EventAppender   EventAppender
	Packer          *envelope.Packer
	Logger          *zap.Logger

	version uint32
	root    dogma.AggregateRoot
}

func (s *InstanceSupervisor) ExecuteCommand(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	cmd, err := s.Packer.Unpack(env)
	if err != nil {
		return err
	}

	rev := &RevisionRecord{}

	s.Handler.HandleCommand(
		s.root,
		&scope{
			HandlerIdentity: s.HandlerIdentity,
			InstID:          s.InstanceID,
			Root:            s.root,
			Packer:          s.Packer,
			Logger:          s.Logger.With(zapx.Envelope("command", env)),
			CommandEnvelope: env,
			Revision:        rev,
		},
		cmd,
	)

	r := &JournalRecord_Revision{
		Revision: rev,
	}

	if err := s.apply(ctx, r); err != nil {
		return err
	}

	var envelopes []*envelopespec.Envelope
	for _, a := range rev.Actions {
		if r := a.GetRecordEvent(); r != nil {
			envelopes = append(envelopes, r.Envelope)
		}
	}

	if len(envelopes) == 0 {
		return nil
	}

	return s.EventAppender.Append(ctx, envelopes...)
}

func (s *InstanceSupervisor) applyRevision(r *RevisionRecord) {
}

// load reads all entries from the journal and applies them to the instance.
func (s *InstanceSupervisor) load(ctx context.Context) error {
	if s.root != nil {
		return nil
	}

	s.root = s.Handler.New()

	for {
		r, ok, err := s.Journal.Read(ctx, s.version)
		if err != nil {
			return fmt.Errorf("unable to load instance: %w", err)
		}
		if !ok {
			break
		}

		r.GetOneOf().(journalRecord).apply(s)
		s.version++
	}

	s.Logger.Debug(
		"loaded aggregate instance from journal",
		zap.Uint32("version", s.version),
	)

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (s *InstanceSupervisor) apply(
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
	apply(s *InstanceSupervisor)
}

func (x *JournalRecord_Revision) apply(s *InstanceSupervisor) { s.applyRevision(x.Revision) }
