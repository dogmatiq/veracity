package journal

import (
	"context"

	"github.com/dogmatiq/veracity/journal/internal/indexpb"
	"github.com/dogmatiq/veracity/persistence"
)

func queueNodeKey(messageID string) string {
	return persistence.Key("queue", messageID)
}

func (c *Committer) commandEnqueued(ctx context.Context, id []byte, rec *CommandEnqueued) error {
	messageID := rec.Envelope.MessageId

	node := &indexpb.QueueNode{
		Envelope:      rec.Envelope,
		PrevMessageId: c.metaData.QueueTailMessageId,
	}

	// Mark the new messaqe as the new tail.
	c.metaData.QueueTailMessageId = messageID

	if c.metaData.QueueHeadMessageId == "" {
		// The queue is empty, this message becomes both the head and the tail.
		c.metaData.QueueHeadMessageId = messageID
	} else {
		// The queue is not empty, load the current tail message so we can link
		// it to this message.
		prev := &indexpb.QueueNode{}
		if err := c.update(
			ctx,
			queueNodeKey(node.PrevMessageId),
			prev,
			func() {
				prev.NextMessageId = messageID
			},
		); err != nil {
			return err
		}
	}

	return c.set(
		ctx,
		queueNodeKey(messageID),
		node,
	)
}

func (c *Committer) commandAcknowledged(ctx context.Context, messageID string) error {
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
	return c.remove(ctx, key)
}
