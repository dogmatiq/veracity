package instjournal

import (
	"context"

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
		"persistence.journal",
		telemetry.String("name", name),
	)

	ctx, span := r.StartSpan(ctx, "JournalStore.Open")
	defer span.End()

	m := newMetrics(r)

	j, err := s.Store.Open(ctx, name)
	if err != nil {
		span.Error("unable to open journal", err)
		return nil, err
	}

	span.Debug("journal opened")
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
		span.Error("unable to read journal record", err)
		return nil, false, err
	}

	if !ok {
		span.Debug("journal record does not exist")
		return nil, false, nil
	}

	size := int64(len(rec))

	span.SetAttributes(
		telemetry.Int("record.size", size),
	)

	j.Metrics.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
	j.Metrics.DataIO.Add(ctx, size, telemetry.ReadDirection)
	j.Metrics.RecordSize.Record(ctx, size, telemetry.ReadDirection)

	span.Debug(
		"read journal record",
		telemetry.Int("record.size", size),
	)

	return rec, true, nil
}

func (j *journ) Range(
	ctx context.Context,
	ver uint64,
	fn journal.RangeFunc,
) error {
	return j.instrumentRange(
		ctx,
		"Journal.Range",
		ver,
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
	return j.instrumentRange(
		ctx,
		"Journal.RangeAll",
		0,
		fn,
		j.Journal.RangeAll,
	)
}

func (j *journ) instrumentRange(
	ctx context.Context,
	name string,
	begin uint64,
	fn journal.RangeFunc,
	doRange func(context.Context, journal.RangeFunc) error,
) error {
	ctx, span := j.Telemetry.StartSpan(ctx, name)
	defer span.End()

	var (
		end        = begin
		exhaustive = true
		totalSize  int64
	)

	err := doRange(
		ctx,
		func(ctx context.Context, ver uint64, rec []byte) (bool, error) {
			end++
			size := int64(len(rec))
			totalSize += size

			span.Debug(
				"read journal record",
				telemetry.Int("version", ver),
				telemetry.Int("record.size", size),
			)

			j.Metrics.RecordIO.Add(ctx, 1, telemetry.ReadDirection)
			j.Metrics.DataIO.Add(ctx, size, telemetry.ReadDirection)
			j.Metrics.RecordSize.Record(ctx, size, telemetry.ReadDirection)

			ok, err := fn(ctx, ver, rec)
			if !ok || err != nil {
				exhaustive = false
			}
			return ok, err
		},
	)

	span.SetAttributes(
		telemetry.Int("version.range_begin_inclusive", begin),
		telemetry.Int("version.range_end_exclusive", end),
		telemetry.Bool("version.range_exhaustive", exhaustive),
		telemetry.Int("record.count", end-begin),
		telemetry.Int("record.total_size", totalSize),
	)

	if err != nil {
		span.Error("unable to range over records", err)
	}

	return err
}

func (j *journ) Append(ctx context.Context, ver uint64, rec []byte) (bool, error) {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Append",
		telemetry.Int("version", ver),
		telemetry.Int("record.size", len(rec)),
	)
	defer span.End()

	ok, err := j.Journal.Append(ctx, ver, rec)
	if err != nil {
		span.Error("unable to append journal record", err)
		return false, err
	}

	span.SetAttributes(
		telemetry.Bool("version.conflict", !ok),
	)

	if ok {
		span.Debug("journal record appended")
	} else {
		j.Metrics.ConflictCount.Add(ctx, 1)
		span.Debug("optimistic concurrency conflict in journal")
	}

	return ok, nil
}

func (j *journ) Truncate(ctx context.Context, ver uint64) error {
	ctx, span := j.Telemetry.StartSpan(
		ctx,
		"Journal.Truncate",
		telemetry.Int("version.range_begin_inclusive", 0),
		telemetry.Int("version.range_end_exclusive", ver),
	)
	defer span.End()

	if err := j.Journal.Truncate(ctx, ver); err != nil {
		span.Error("unable to truncate journal records", err)
		return err
	}

	span.Debug("journal records truncated")

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
		span.Error("unable to close journal", err)
		return err
	}

	span.Debug("journal closed")

	return nil
}
