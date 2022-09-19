package aggregate

import (
	"context"
	"errors"
	"fmt"

	"github.com/dogmatiq/dogma"
	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/internal/envelope"
	"github.com/dogmatiq/veracity/internal/fsm"
	"github.com/dogmatiq/veracity/internal/protojournal"
	"github.com/dogmatiq/veracity/internal/zapx"
	"github.com/dogmatiq/veracity/persistence/journal"
	"go.uber.org/zap"
)

// instance executes commands against a single aggregate instance.
type instance struct {
	// HandlerIdentity is the identity of the handler used to handle commands.
	HandlerIdentity *envelopespec.Identity

	// Handler is the handler used to handle commands.
	Handler dogma.AggregateMessageHandler

	// InstanceID is the aggregate instance ID.
	InstanceID string

	// Requests is a channel that receives command execution requests. It is
	// closed when the instance can be unloaded.
	Requests <-chan request

	// Packer is used to pack envelopes that contain the domain events that are
	// recorded by the handler.
	Packer *envelope.Packer

	// Journal is the journal used to store the instance's state.
	Journal journal.Journal

	// EventAppender is used to append events to the global event stream.
	EventAppender EventAppender

	// Logger is the target for messages about the execution of commands and
	// management of aggregate state.
	Logger *zap.Logger

	version  uint64
	commands map[string]struct{}
	pending  *RevisionRecord
	root     dogma.AggregateRoot
}

// Run executes commands against the aggregate instance until ctx is canceled,
// the Requests channel is closed or an error occurs.
func (i *instance) Run(ctx context.Context) error {
	if err := i.load(ctx); err != nil {
		return err
	}

	return fsm.Run(ctx, i.stateAwait)
}

// stateAwait waits for a command request to be received.
func (i *instance) stateAwait(ctx context.Context) (fsm.Action, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case req := <-i.Requests:
			return fsm.TransitionWith(i.stateHandle, req), nil
		}
	}
}

// stateHandle handles a command request.
func (i *instance) stateHandle(ctx context.Context, req request) (fsm.Action, error) {
	if err := i.executeCommand(ctx, req.CommandEnvelope); err != nil {
		return nil, err
	}

	close(req.Done)

	return fsm.Transition(i.stateAwait), nil
}

// executeCommand executes a command against the aggregate instance.
func (i *instance) executeCommand(ctx context.Context, env *envelopespec.Envelope) error {
	if i.pending != nil {
		panic("cannot handle command while last revision is pending")
	}

	if _, ok := i.commands[env.MessageId]; ok {
		i.Logger.Debug(
			"ignored duplicate command",
			zapx.Envelope("command", env),
		)

		return nil
	}

	cmd, err := i.Packer.Unpack(env)
	if err != nil {
		return fmt.Errorf("unable to unpack command from envelope: %w", err)
	}

	rev := &RevisionRecord{
		CommandId: env.MessageId,
	}

	sc := &scope{
		HandlerIdentity: i.HandlerIdentity,
		InstID:          i.InstanceID,
		Root:            i.root,
		Packer:          i.Packer,
		Logger:          i.Logger,
		CommandEnvelope: env,
		Revision:        rev,
	}

	i.Handler.HandleCommand(i.root, sc, cmd)

	rec := &JournalRecord_Revision{
		Revision: rev,
	}

	if err := i.append(ctx, rec); err != nil {
		return fmt.Errorf("unable to record revision: %w", err)
	}

	i.commands[env.MessageId] = struct{}{}

	return i.EventAppender.Append(ctx, sc.Revision.EventEnvelopes...)
}

// apply updates the instance's in-memory state to reflect a revision record.
func (x *JournalRecord_Revision) apply(i *instance) error {
	i.commands[x.Revision.CommandId] = struct{}{}
	i.pending = x.Revision

	for _, env := range i.pending.EventEnvelopes {
		event, err := i.Packer.Unpack(env)
		if err != nil {
			return fmt.Errorf("unable to unmarshal historical event: %w", err)
		}

		i.root.ApplyEvent(event)
	}

	return nil
}

// load reads all records from the journal and applies them to the instance.
func (i *instance) load(ctx context.Context) error {
	i.commands = map[string]struct{}{}
	i.root = i.Handler.New()
	rec := &JournalRecord{}

	for {
		ok, err := protojournal.Get(ctx, i.Journal, i.version, rec)
		if err != nil {
			return fmt.Errorf("unable to load instance: %w", err)
		}
		if !ok {
			break
		}

		type applyer interface {
			apply(*instance) error
		}
		if err := rec.GetOneOf().(applyer).apply(i); err != nil {
			return fmt.Errorf("unable to load instance: %w", err)
		}

		i.version++
	}

	if i.pending != nil {
		if err := i.EventAppender.Append(ctx, i.pending.EventEnvelopes...); err != nil {
			return err
		}

		i.pending = nil
	}

	i.Logger.Debug(
		"loaded aggregate instance",
		zap.Uint64("version", i.version),
	)

	return nil
}

// append adds a record to the journal.
func (i *instance) append(ctx context.Context, rec isJournalRecord_OneOf) error {
	ok, err := protojournal.Append(
		ctx,
		i.Journal,
		i.version,
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

	i.version++

	return nil
}
