package instrumentedpersistence

import (
	"context"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/persistence/journal"
	"go.opentelemetry.io/otel/metric"
)

// JournalStore is a decorator that adds instrumentation to a [journal.Store].
type JournalStore struct {
	Next      journal.Store
	Telemetry *telemetry.Provider
}

// Open returns the journal with the given name.
func (s *JournalStore) Open(ctx context.Context, name string) (journal.Journal, error) {
	r := s.Telemetry.Recorder(
		"github.com/dogmatiq/veracity/persistence",
		"journal",
		telemetry.Type("store", s.Next),
		telemetry.String("handle", handleID()),
		telemetry.String("name", name),
	)

	ctx, span := r.StartSpan(ctx, "journal.open")
	defer span.End()

	next, err := s.Next.Open(ctx, name)
	if err != nil {
		span.Error("could not open journal", err)
		return nil, err
	}

	j := &journ{
		Next:      next,
		Telemetry: r,
		OpenCount: r.Int64UpDownCounter(
			"open_journals",
			metric.WithDescription("The number of journals that are currently open."),
			metric.WithUnit("{journal}"),
		),
		ConflictCount: r.Int64Counter(
			"conflicts",
			metric.WithDescription("The number of times appending a record to the journal has failed due to a optimistic-concurrency conflict."),
			metric.WithUnit("{conflict}"),
		),
		DataIO: r.Int64Counter(
			"io",
			metric.WithDescription("The cumulative size of the journal records that have been read and written."),
			metric.WithUnit("By"),
		),
		RecordIO: r.Int64Counter(
			"record.io",
			metric.WithDescription("The number of journal records that have been read and written."),
			metric.WithUnit("{record}"),
		),
		RecordSize: r.Int64Histogram(
			"record.size",
			metric.WithDescription("The sizes of the journal records that have been read and written."),
			metric.WithUnit("By"),
		),
	}

	j.OpenCount.Add(ctx, 1)
	span.Debug("opened journal")

	return j, nil
}

type journ struct {
	Next      journal.Journal
	Telemetry *telemetry.Recorder

	OpenCount     metric.Int64UpDownCounter
	ConflictCount metric.Int64Counter
	DataIO        metric.Int64Counter
	RecordIO      metric.Int64Counter
	RecordSize    metric.Int64Histogram
}

func (j *journ) Get(ctx context.Context, offset uint64) (_ []byte, ok bool, err error) {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"journal.get",
		telemetry.Int("offset", offset),
	)
	defer span.End()

	rec, ok, err := j.Next.Get(ctx, offset)
	if err != nil {
		span.Error("could not fetch journal record", err)
		return nil, false, err
	}

	if !ok {
		span.Debug("could not fetch non-existent journal record")
		return nil, false, nil
	}

	size := int64(len(rec))

	span.SetAttributes(
		telemetry.Int("record_size", size),
	)

	j.DataIO.Add(ctx, size, telemetry.ReadDirection)
	j.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
	j.RecordSize.Record(ctx, size, telemetry.ReadDirection)

	span.Debug("fetched single journal record")

	return rec, true, nil
}

func (j *journ) Range(
	ctx context.Context,
	begin uint64,
	fn journal.RangeFunc,
) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"journal.range",
		telemetry.Int("range_start", begin),
	)
	defer span.End()

	return j.instrumentRange(
		ctx,
		span,
		fn,
		func(ctx context.Context, fn journal.RangeFunc) error {
			return j.Next.Range(ctx, begin, fn)
		},
	)
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	ctx, span := j.Telemetry.StartSpan(ctx, "journal.range_all")
	defer span.End()

	return j.instrumentRange(
		ctx,
		span,
		fn,
		j.Next.RangeAll,
	)
}

func (j *journ) instrumentRange(
	ctx context.Context,
	span *telemetry.Span,
	fn journal.RangeFunc,
	doRange func(context.Context, journal.RangeFunc) error,
) error {
	var (
		first, count uint64
		totalSize    int64
		brokeLoop    bool
	)

	span.Debug("reading journal records")

	err := doRange(
		ctx,
		func(ctx context.Context, offset uint64, rec []byte) (bool, error) {
			if count == 0 {
				first = offset
			}
			count++

			size := int64(len(rec))
			totalSize += size

			j.DataIO.Add(ctx, size, telemetry.ReadDirection)
			j.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
			j.RecordSize.Record(ctx, size, telemetry.ReadDirection)

			ok, err := fn(ctx, offset, rec)
			if ok || err != nil {
				return ok, err
			}

			brokeLoop = true
			return false, nil
		},
	)

	if count != 0 {
		span.SetAttributes(
			telemetry.Int("range_start", first),
			telemetry.Int("range_stop", first+count-1),
		)
	}

	span.SetAttributes(
		telemetry.Int("record_read", count),
		telemetry.Int("bytes_read", totalSize),
		telemetry.Bool("reached_end", !brokeLoop && err == nil),
	)

	if err != nil {
		span.Error("could not read journal records", err)
		return err
	}

	span.Debug("completed reading journal records")

	return nil
}

func (j *journ) Append(ctx context.Context, offset uint64, rec []byte) (bool, error) {
	size := int64(len(rec))

	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"journal.append",
		telemetry.Int("offset", offset),
		telemetry.Int("record_size", size),
	)
	defer span.End()

	j.DataIO.Add(ctx, size, telemetry.WriteDirection)
	j.RecordIO.Add(ctx, 1, telemetry.WriteDirection)
	j.RecordSize.Record(ctx, size, telemetry.WriteDirection)

	ok, err := j.Next.Append(ctx, offset, rec)
	if err != nil {
		span.Error("unable to append journal record", err)
		return false, err
	}

	if ok {
		span.Debug("journal record appended")
	} else {
		span.SetAttributes(
			telemetry.Bool("conflict", true),
		)

		j.ConflictCount.Add(ctx, 1)

		span.Error("journal record conflict", nil)
	}

	return ok, nil
}

func (j *journ) Truncate(ctx context.Context, end uint64) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"journal.truncate",
		telemetry.Int("retained_offset", end),
	)
	defer span.End()

	if err := j.Next.Truncate(ctx, end); err != nil {
		span.Error("unable to truncate journal", err)
		return err
	}

	span.Debug("truncated oldest journal records")

	return nil
}

func (j *journ) Close() (err error) {
	ctx, span := j.Telemetry.StartSpan(context.Background(), "journal.close")
	defer span.End()

	if j.Next == nil {
		span.Warn("journal is already closed")
		return nil
	}

	defer func() {
		j.Next = nil
		j.OpenCount.Add(ctx, -1)
	}()

	if err := j.Next.Close(); err != nil {
		span.Error("could not close journal", err)
		return err
	}

	span.Debug("closed journal")

	return nil
}
