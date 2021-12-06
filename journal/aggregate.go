package journal

import (
	"context"

	"github.com/dogmatiq/veracity/journal/internal/indexpb"
	"github.com/dogmatiq/veracity/persistence"
)

func aggregateInstanceMetaDataKey(handlerKey, instanceID string) string {
	return persistence.Key("aggregate", handlerKey, instanceID, "meta-data")
}

func (c *Committer) commandHandledByAggregate(
	ctx context.Context,
	id []byte,
	rec *CommandHandledByAggregate,
) error {
	if err := c.commandAcknowledged(ctx, rec.MessageId); err != nil {
		return err
	}

	metaData := &indexpb.AggregateInstanceMetaData{}
	if err := c.update(
		ctx,
		aggregateInstanceMetaDataKey(rec.Handler.Key, rec.Instance.Id),
		metaData,
		func() {
			// If this is the first revision of this aggregate, or it has been
			// destroyed, this record is now the aggregate's first relevant
			// record.
			if len(metaData.HeadRecordId) == 0 || rec.Instance.Destroyed {
				metaData.HeadRecordId = id
			}
		},
	); err != nil {
		return err
	}

	return nil
}
