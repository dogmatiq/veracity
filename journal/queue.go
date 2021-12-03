package journal

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
	"github.com/dogmatiq/veracity/journal/internal/indexpb"
	"github.com/dogmatiq/veracity/persistence"
)

// queueNodeKey returns the key used to store the envelope for the given
// message ID on the queue.
func queueNodeKey(messageID string) string {
	return persistence.Key("queue", messageID)
}

// addMessageToQueue adds a message to the queue.
func (c *Committer) addMessageToQueue(ctx context.Context, env *envelopespec.Envelope) error {
	node := &indexpb.QueueNode{
		Envelope:      env,
		PrevMessageId: c.metaData.QueueTailMessageId,
	}

	// Mark the new messaqe as the new tail.
	c.metaData.QueueTailMessageId = env.MessageId

	if c.metaData.QueueHeadMessageId == "" {
		// The queue is empty, this message becomes both the head and the tail.
		c.metaData.QueueHeadMessageId = env.MessageId
	} else {
		// The queue is not empty, load the current tail message so we can link
		// it to this message.
		prev := &indexpb.QueueNode{}
		if err := c.update(
			ctx,
			queueNodeKey(node.PrevMessageId),
			prev,
			func() {
				prev.NextMessageId = env.MessageId
			},
		); err != nil {
			return err
		}
	}

	return c.set(
		ctx,
		queueNodeKey(env.MessageId),
		node,
	)
}

// removeMessageFromQueue removes a message from the queue.
func (c *Committer) removeMessageFromQueue(ctx context.Context, messageID string) error {
	key := queueNodeKey(messageID)
	node := &indexpb.QueueNode{}

	if err := c.get(ctx, key, node); err != nil {
		return err
	}

	// If this node has a node before it, update that node to point to the
	// node after this one.
	if node.PrevMessageId != "" {
		prev := &indexpb.QueueNode{}

		if err := c.update(
			ctx,
			queueNodeKey(node.PrevMessageId),
			prev,
			func() {
				prev.NextMessageId = node.NextMessageId
			},
		); err != nil {
			return err
		}
	}

	// If this node has a node after it, update that node's reverse pointer to
	// point to the node before this one.
	if node.NextMessageId != "" {
		next := &indexpb.QueueNode{}

		if err := c.update(
			ctx,
			queueNodeKey(node.NextMessageId),
			next,
			func() {
				next.PrevMessageId = node.PrevMessageId
			},
		); err != nil {
			return err
		}
	}

	// If this node was the head of the queue, the new head is the next node, if
	// any.
	if c.metaData.QueueHeadMessageId == messageID {
		c.metaData.QueueHeadMessageId = node.NextMessageId
	}

	// If this node was the tail of the queue, the new tail is the previous
	// node, if any.
	if c.metaData.QueueTailMessageId == messageID {
		c.metaData.QueueTailMessageId = node.PrevMessageId
	}

	// Only after all of the surrounding nodes have been updated successfully do
	// we actually delete this node, otherwise we lose the references to the
	// next/previous nodes required above.
	return c.Index.Set(ctx, key, nil)
}
