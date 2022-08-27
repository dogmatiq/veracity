package aggregate

import (
	"context"

	"golang.org/x/exp/slices"
)

type Loader struct {
	Journal Journal
	Stream  EventStream
}

func (l *Loader) Load(
	ctx context.Context,
	hk, id string,
	sn *Snapshot,
) error {
	var offset uint64
	for {
		records, err := l.Journal.Read(ctx, hk, id, offset)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}

		for _, r := range records {
			offset++
			r.ApplyTo(sn)
		}
	}

	offset = 0
	for len(sn.PendingEvents) != 0 {
		events, err := l.Stream.Read(ctx, offset)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			break
		}

		for _, ev := range events {
			offset++

			for i, unpublished := range sn.PendingEvents {
				if unpublished.ID() == ev.ID() {
					sn.PendingEvents = slices.Delete(sn.PendingEvents, i, i+1)
					break
				}
			}
		}
	}

	return nil
}
