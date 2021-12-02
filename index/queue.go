package index

import (
	"context"

	"github.com/dogmatiq/interopspec/envelopespec"
)

// queueKey is the key used to store the queue itself.
var queueKey = formatKey("queue")

// queueMessageKey returns the key used to store the envelope for the given
// message ID on the queue.
func queueMessageKey(messageID string) []byte {
	return formatKey("%s/message/%s", queueKey, messageID)
}

// addMessageToQueue adds a message to the queue.
func addMessageToQueue(
	ctx context.Context,
	index Index,
	env *envelopespec.Envelope,
) error {
	// Store the message envelope.
	if err := setProto(
		ctx,
		index,
		queueMessageKey(env.MessageId),
		env,
	); err != nil {
		return err
	}

	queue := &Queue{}
	if err := getProto(ctx, index, queueKey, queue); err != nil {
		return err
	}

	// Bail if the message ID is already in the queue.
	for _, id := range queue.MessageIds {
		if id == env.MessageId {
			return nil
		}
	}

	// Add the message ID to the queue.
	queue.MessageIds = append(queue.MessageIds, env.MessageId)

	return setProto(ctx, index, queueKey, queue)
}

// removeMessageFromQueue removes a message from the queue.
func removeMessageFromQueue(
	ctx context.Context,
	index Index,
	messageID string,
) error {
	// Delete the message envelope.
	if err := index.Set(
		ctx,
		queueMessageKey(messageID),
		nil,
	); err != nil {
		return err
	}

	// Load the queue itself.
	queue := &Queue{}
	if err := getProto(ctx, index, queueKey, queue); err != nil {
		return err
	}

	// Remove the message ID from the queue.
	for i, id := range queue.MessageIds {
		if id == messageID {
			n := copy(queue.MessageIds[i:], queue.MessageIds[i+1:])
			queue.MessageIds = queue.MessageIds[:n]
			return setProto(ctx, index, queueKey, queue)
		}
	}

	// The message ID was not in the queue.
	return nil
}
