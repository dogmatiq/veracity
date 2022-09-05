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

type instance struct {
	HandlerIdentity *envelopespec.Identity
	InstanceID      string
	Handler         dogma.AggregateMessageHandler
	Journal         journal.Journal[*JournalRecord]
	EventAppender   EventAppender
	Packer          *envelope.Packer
	Logger          *zap.Logger

	version     uint32
	commands    map[string]struct{}
	unpublished []*envelopespec.Envelope
	root        dogma.AggregateRoot
}

func (s *instance) ExecuteCommand(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	if err := s.load(ctx); err != nil {
		return err
	}

	if len(s.unpublished) != 0 {
		panic("cannot handle command while there are unpublished events")
	}

	if _, ok := s.commands[env.MessageId]; ok {
		return nil
	}

	cmd, err := s.Packer.Unpack(env)
	if err != nil {
		return err
	}

	rev := &RevisionRecord{
		CommandId: env.MessageId,
	}

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

	if len(s.unpublished) == 0 {
		return nil
	}

	if err := s.EventAppender.Append(ctx, s.unpublished...); err != nil {
		return err
	}

	s.unpublished = nil

	return nil
}

func (s *instance) applyRevision(r *RevisionRecord) {
	s.commands[r.CommandId] = struct{}{}
	s.unpublished = nil

	for _, a := range r.Actions {
		if r := a.GetRecordEvent(); r != nil {
			s.unpublished = append(
				s.unpublished,
				r.Envelope,
			)
		}
	}
}

// load reads all entries from the journal and applies them to the instance.
func (s *instance) load(ctx context.Context) error {
	if s.root != nil {
		return nil
	}

	s.commands = map[string]struct{}{}
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

	if len(s.unpublished) != 0 {
		if err := s.EventAppender.Append(ctx, s.unpublished...); err != nil {
			return err
		}
		s.unpublished = nil
	}

	s.Logger.Debug(
		"loaded aggregate instance from journal",
		zap.Uint32("version", s.version),
	)

	return nil
}

// apply writes a record to the journal and applies it to the queue.
func (s *instance) apply(
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
	apply(s *instance)
}

func (x *JournalRecord_Revision) apply(s *instance) { s.applyRevision(x.Revision) }
