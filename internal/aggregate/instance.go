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
	"github.com/dogmatiq/veracity/persistence/kv"
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

	// Keyspace is the key-value store used to store the instance's state.
	Keyspace kv.Keyspace

	// EventAppender is used to append events to the global event stream.
	EventAppender EventAppender

	// Logger is the target for messages about the execution of commands and
	// management of aggregate state.
	Logger *zap.Logger

	version     uint64
	unpublished *RevisionRecord
	root        dogma.AggregateRoot
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
	if i.unpublished != nil {
		panic("cannot handle command while last revision is pending")
	}

	handled, err := i.hasHandledMessage(ctx, env.MessageId)
	if err != nil {
		return fmt.Errorf("Unable to check if command has been handled: %w", err)
	}
	if handled {
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

	return i.publishRevision(ctx, rev)
}

// apply updates the instance's in-memory state to reflect a revision record.
func (x *JournalRecord_Revision) apply(i *instance) error {
	i.unpublished = x.Revision

	for _, env := range i.unpublished.EventEnvelopes {
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
	i.root = i.Handler.New()

	if err := protojournal.RangeAll(
		ctx,
		i.Journal,
		func(
			ctx context.Context,
			ver uint64,
			rec *JournalRecord,
		) (bool, error) {
			i.version = ver + 1

			type applyer interface {
				apply(*instance) error
			}
			return true, rec.GetOneOf().(applyer).apply(i)
		},
	); err != nil {
		return fmt.Errorf("unable to load instance: %w", err)
	}

	if i.unpublished != nil {
		if err := i.publishRevision(ctx, i.unpublished); err != nil {
			return fmt.Errorf("unable to load instance: %w", err)
		}

		i.unpublished = nil
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

// publishRevision makes the result of a revision known outside the instance's
// own journal.
func (i *instance) publishRevision(ctx context.Context, rev *RevisionRecord) error {
	if err := i.EventAppender.Append(ctx, rev.EventEnvelopes...); err != nil {
		return err
	}

	if err := i.markCommandAsHandled(ctx, rev.CommandId); err != nil {
		return err
	}

	return nil
}

// hasHandledMessage returns true if the command with the given ID has already
// been handled.
func (i *instance) hasHandledMessage(
	ctx context.Context,
	id string,
) (bool, error) {
	return i.Keyspace.Has(
		ctx,
		commandKey(id),
	)
}

var trueValue = []byte{1}

// markCommandAsHandled marks the command with the given ID as handled.
func (i *instance) markCommandAsHandled(
	ctx context.Context,
	id string,
) error {
	if err := i.Keyspace.Set(
		ctx,
		commandKey(id),
		trueValue,
	); err != nil {
		return fmt.Errorf("unable to mark command as handled: %w", err)
	}

	return nil
}

// commandKey returns the keyspace key used to store a value that indicates that
// the command with the given ID has been handled.
func commandKey(id string) []byte {
	return []byte("command." + id)
}
