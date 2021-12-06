package journal

import "context"

func (c *Committer) eventHandledByProcess(ctx context.Context, id []byte, rec *EventHandledByProcess) error {
	return nil
}

func (c *Committer) timeoutHandledByProcess(ctx context.Context, id []byte, rec *TimeoutHandledByProcess) error {
	return nil
}
