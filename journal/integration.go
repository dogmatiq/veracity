package journal

import "context"

func (c *Committer) commandHandledByIntegration(
	ctx context.Context,
	id []byte,
	rec *CommandHandledByIntegration,
) error {
	if err := c.commandAcknowledged(ctx, rec.MessageId); err != nil {
		return err
	}

	return nil
}
