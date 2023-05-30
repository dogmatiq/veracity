package instjournal

import (
	"context"
	"errors"

	"github.com/dogmatiq/veracity/internal/telemetry"
	"github.com/dogmatiq/veracity/persistence/journal"
)

// Store is a decorator that adds instrumentation to a [journal.JournalStore].
type Store struct {
	Store     journal.Store
	Telemetry *telemetry.Provider
}

// Open returns the journal with the given name.
func (s *Store) Open(ctx context.Context, name string) (_ journal.Journal, err error) {
	r := s.Telemetry.New(
		"github.com/dogmatiq/veracity/persistence",
		"journal",
		telemetry.String("name", name),
	)

	ctx, span := r.StartSpan(ctx, "JournalStore.Open")
	defer span.End()

	m := newMetrics(r)

	j, err := s.Store.Open(ctx, name)
	if err != nil {
		span.Error("could not open journal", err)
		return nil, err
	}

	span.Debug("opened journal")
	m.OpenCount.Add(ctx, 1)

	return &journ{
		Journal:   j,
		Telemetry: r,
		Metrics:   m,
	}, nil
}

type journ struct {
	Journal   journal.Journal
	Telemetry *telemetry.Recorder
	Metrics   metrics
}

func (j *journ) Get(ctx context.Context, ver uint64) (_ []byte, ok bool, err error) {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Get",
		telemetry.Int("version", ver),
	)
	defer span.End()

	rec, ok, err := j.Journal.Get(ctx, ver)
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

	j.Metrics.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
	j.Metrics.DataIO.Add(ctx, size, telemetry.ReadDirection)
	j.Metrics.RecordSize.Record(ctx, size, telemetry.ReadDirection)

	span.Debug("fetched single journal record")

	return rec, true, nil
}

func (j *journ) Range(
	ctx context.Context,
	ver uint64,
	fn journal.RangeFunc,
) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Range",
		telemetry.Int("range_begin", ver),
	)
	defer span.End()

	return j.instrumentRange(
		ctx,
		span,
		fn,
		func(ctx context.Context, fn journal.RangeFunc) error {
			return j.Journal.Range(ctx, ver, fn)
		},
	)
}

func (j *journ) RangeAll(
	ctx context.Context,
	fn journal.RangeFunc,
) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.RangeAll",
	)
	defer span.End()

	return j.instrumentRange(
		ctx,
		span,
		fn,
		j.Journal.RangeAll,
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
		func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
			if count == 0 {
				first = ver
			}
			count++

			size := int64(len(rec))
			totalSize += size

			j.Metrics.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
			j.Metrics.DataIO.Add(ctx, size, telemetry.ReadDirection)
			j.Metrics.RecordSize.Record(ctx, size, telemetry.ReadDirection)

			ok, err := fn(ctx, ver, rec)
			if ok || err != nil {
				return ok, err
			}

			brokeLoop = true
			return false, nil
		},
	)

	if count != 0 {
		span.SetAttributes(
			telemetry.Int("range_begin", first),
			telemetry.Int("range_end", first+count-1),
		)
	}

	span.SetAttributes(
		telemetry.Bool("reached_final_record", !brokeLoop && err == nil),
		telemetry.Int("record_size_total", totalSize),
		telemetry.Int("records_iterated", count),
	)

	if err != nil {
		span.Error("could not read journal records", err)
		return err
	}

	span.Debug("completed reading journal records")
	return nil
}

func (j *journ) Append(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	size := int64(len(rec))

	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Append",
		telemetry.Int("version", ver),
		telemetry.Int("record_size", size),
	)
	defer span.End()

	j.Metrics.RecordIO.Add(ctx, 1, telemetry.WriteDirection)
	j.Metrics.DataIO.Add(ctx, size, telemetry.WriteDirection)
	j.Metrics.RecordSize.Record(ctx, size, telemetry.WriteDirection)

	ok, err := j.Journal.Append(ctx, ver, rec)
	if err != nil {
		span.Error("unable to append journal record", err)
		return false, err
	}

	span.SetAttributes(
		telemetry.Bool("conflict", !ok),
	)

	if ok {
		span.Debug("journal record appended")
	} else {
		j.Metrics.ConflictCount.Add(ctx, 1)
		span.Error(
			"journal version conflict",
			errors.New("journal version conflict"),
		)
	}

	return ok, nil
}

func (j *journ) Truncate(ctx context.Context, ver uint64) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Truncate",
		telemetry.Int("oldest_retained_version", ver),
	)
	defer span.End()

	if err := j.Journal.Truncate(ctx, ver); err != nil {
		span.Error("unable to truncate journal", err)
		return err
	}

	span.Debug("truncated oldest journal records")

	return nil
}

func (j *journ) Close() (err error) {
	ctx, span := j.Telemetry.StartSpan(context.Background(), "Journal.Close")
	defer span.End()

	if j.Journal == nil {
		span.Warn("journal is already closed")
		return nil
	}

	defer func() {
		j.Journal = nil
		j.Metrics.OpenCount.Add(ctx, -1)
	}()

	if err := j.Journal.Close(); err != nil {
		span.Error("could not close journal", err)
		return err
	}

	span.Debug("closed journal")

	return nil
}
