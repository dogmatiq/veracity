package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/journal"
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
	Requests        <-chan request

	version     uint32
	commands    map[string]struct{}
	unpublished []*envelopespec.Envelope
	root        dogma.AggregateRoot
}

func (i *instance) Run(ctx context.Context) error {
	if err := i.load(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case req := <-i.Requests:
			err := i.executeCommand(ctx, req.CommandEnvelope)
			req.Response <- err
			if err != nil {
				return err
			}
		}
	}
}

func (i *instance) executeCommand(
	ctx context.Context,
	env *envelopespec.Envelope,
) error {
	if len(i.unpublished) != 0 {
		panic("cannot handle command while there are unpublished events")
	}

	if _, ok := i.commands[env.MessageId]; ok {
		return nil
	}

	cmd, err := i.Packer.Unpack(env)
	if err != nil {
		return err
	}

	rev := &RevisionRecord{
		CommandId: env.MessageId,
	}

	i.Handler.HandleCommand(
		i.root,
		&scope{
			HandlerIdentity: i.HandlerIdentity,
			InstID:          i.InstanceID,
			Root:            i.root,
			Packer:          i.Packer,
			Logger:          i.Logger.With(zapx.Envelope("command", env)),
			CommandEnvelope: env,
			Revision:        rev,
		},
		cmd,
	)

	r := &JournalRecord_Revision{
		Revision: rev,
	}

	if err := i.apply(ctx, r); err != nil {
		return fmt.Errorf("unable to record revision: %w", err)
	}

	if len(i.unpublished) == 0 {
		return nil
	}

	if err := i.EventAppender.Append(ctx, i.unpublished...); err != nil {
		return err
	}

	i.unpublished = nil

	return nil
}

func (i *instance) applyRevision(r *RevisionRecord) {
	i.commands[r.CommandId] = struct{}{}
	i.unpublished = nil

	for _, a := range r.Actions {
		if r := a.GetRecordEvent(); r != nil {
			i.unpublished = append(
				i.unpublished,
				r.Envelope,
			)
		}
	}
}

// load reads all entries from the journal and applies them to the instance.
func (i *instance) load(ctx context.Context) error {
	i.commands = map[string]struct{}{}
	i.root = i.Handler.New()

	for {
		r, ok, err := i.Journal.Read(ctx, i.version)
		if err != nil {
			return fmt.Errorf("unable to load instance: %w", err)
		}
		if !ok {
			break
		}

		r.GetOneOf().(journalRecord).apply(i)
		i.version++
	}

	if len(i.unpublished) != 0 {
		if err := i.EventAppender.Append(ctx, i.unpublished...); err != nil {
			return err
		}
		i.unpublished = nil
	}

	i.Logger.Debug(
		"loaded aggregate instance from journal",
		zap.Uint32("version", i.version),
	)

	return nil
}

// apply writes a record to the journal and applies it to the instance.
func (i *instance) apply(
	ctx context.Context,
	r journalRecord,
) error {
	ok, err := i.Journal.Write(
		ctx,
		i.version,
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

	r.apply(i)
	i.version++

	return nil
}

type journalRecord interface {
	isJournalRecord_OneOf
	apply(i *instance)
}

func (x *JournalRecord_Revision) apply(i *instance) { i.applyRevision(x.Revision) }
