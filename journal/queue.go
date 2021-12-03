package journal

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/journal/internal/indexpb"
)

// queueKey is the key used to store the queue itself.
var queueKey = makeKey("queue")

// queueMessageKey returns the key used to store the envelope for the given
// message ID on the queue.
func queueMessageKey(messageID string) []byte {
	return makeKey("queue", "message", messageID)
}

// addMessageToQueue adds a message to the queue.
func (c *Committer) addMessageToQueue(ctx context.Context, env *envelopespec.Envelope) error {
	if err := c.setProto(
		ctx,
		queueMessageKey(env.MessageId),
		env,
	); err != nil {
		return err
	}

	queue := &indexpb.Queue{}
	if err := c.getProto(ctx, queueKey, queue); err != nil {
		return err
	}

	for _, id := range queue.MessageIds {
		if id == env.MessageId {
			return nil
		}
	}

	queue.MessageIds = append(queue.MessageIds, env.MessageId)

	return c.setProto(ctx, queueKey, queue)
}

// removeMessageFromQueue removes a message from the queue.
func (c *Committer) removeMessageFromQueue(ctx context.Context, messageID string) error {
	// Delete the message envelope.
	if err := c.Index.Set(
		ctx,
		queueMessageKey(messageID),
		nil,
	); err != nil {
		return err
	}

	// Load the queue itself.
	queue := &indexpb.Queue{}
	if err := c.getProto(ctx, queueKey, queue); err != nil {
		return err
	}

	// Remove the message ID from the queue.
	for i, id := range queue.MessageIds {
		if id == messageID {
			n := copy(queue.MessageIds[i:], queue.MessageIds[i+1:])
			queue.MessageIds = queue.MessageIds[:n]
			return c.setProto(ctx, queueKey, queue)
		}
	}

	// The message ID was not in the queue.
	return nil
}
