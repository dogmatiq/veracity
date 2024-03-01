package instrumentedpersistence

import (
	"context"

	"github.com/dogmatiq/persistencekit/kv"
	"github.com/dogmatiq/veracity/internal/telemetry"
	"go.opentelemetry.io/otel/metric"
)

// KeyValueStore is a decorator that adds instrumentation to a [kv.Store].
type KeyValueStore struct {
	Next      kv.Store
	Telemetry *telemetry.Provider
}

// Open returns the keyspace with the given name.
func (s *KeyValueStore) Open(ctx context.Context, name string) (kv.Keyspace, error) {
	r := s.Telemetry.Recorder(
		"github.com/dogmatiq/veracity/persistence",
		"keyspace",
		telemetry.Type("store", s.Next),
		telemetry.String("handle", handleID()),
		telemetry.String("name", name),
	)

	ctx, span := r.StartSpan(ctx, "keyspace.open")
	defer span.End()

	next, err := s.Next.Open(ctx, name)
	if err != nil {
		span.Error("could not open keyspace", err)
		return nil, err
	}

	ks := &keyspace{
		Next:      next,
		Telemetry: r,
		OpenCount: r.Int64UpDownCounter(
			"open_keyspaces",
			metric.WithDescription("The number of keyspaces that are currently open."),
			metric.WithUnit("{keyspace}"),
		),
		DataIO: r.Int64Counter(
			"io",
			metric.WithDescription("The cumulative size of the keys and values that have been read and written."),
			metric.WithUnit("By"),
		),
		PairIO: r.Int64Counter(
			"pair.io",
			metric.WithDescription("The number of key/value pairs that have been read and written."),
			metric.WithUnit("{pair}"),
		),
		KeySize: r.Int64Histogram(
			"key.size",
			metric.WithDescription("The sizes of the keys that have been read and written."),
			metric.WithUnit("By"),
		),
		ValueSize: r.Int64Histogram(
			"value.size",
			metric.WithDescription("The sizes of the values that have been read and written."),
			metric.WithUnit("By"),
		),
	}

	ks.OpenCount.Add(ctx, 1)
	span.Debug("opened keyspace")

	return ks, nil
}

type keyspace struct {
	Next      kv.Keyspace
	Telemetry *telemetry.Recorder

	OpenCount metric.Int64UpDownCounter
	DataIO    metric.Int64Counter
	PairIO    metric.Int64Counter
	KeySize   metric.Int64Histogram
	ValueSize metric.Int64Histogram
}

func (ks *keyspace) Get(ctx context.Context, k []byte) ([]byte, error) {
	keySize := int64(len(k))

	ctx, span := ks.Telemetry.StartSpan(
		ctx,
		"keyspace.get",
		telemetry.If(
			isShortASCII(k),
			telemetry.String("key", string(k)),
		),
		telemetry.Int("key_size", keySize),
	)
	defer span.End()

	ks.DataIO.Add(ctx, keySize, telemetry.WriteDirection)
	ks.KeySize.Record(ctx, keySize, telemetry.WriteDirection)

	v, err := ks.Next.Get(ctx, k)
	if err != nil {
		span.Error("could not fetch value", err)
		return nil, err
	}

	valueSize := int64(len(v))

	span.SetAttributes(
		telemetry.If(
			isShortASCII(v),
			telemetry.String("value", string(v)),
		),
		telemetry.Int("value_size", valueSize),
	)

	ks.PairIO.Add(ctx, 1, telemetry.ReadDirection)
	ks.DataIO.Add(ctx, valueSize, telemetry.ReadDirection)
	ks.ValueSize.Record(ctx, valueSize, telemetry.ReadDirection)

	span.Debug("fetched value")

	return v, nil
}

func (ks *keyspace) Has(ctx context.Context, k []byte) (bool, error) {
	keySize := int64(len(k))

	ctx, span := ks.Telemetry.StartSpan(
		ctx,
		"keyspace.has",
		telemetry.If(
			isShortASCII(k),
			telemetry.String("key", string(k)),
		),
		telemetry.Int("key_size", keySize),
	)
	defer span.End()

	ks.DataIO.Add(ctx, keySize, telemetry.WriteDirection)
	ks.KeySize.Record(ctx, keySize, telemetry.WriteDirection)

	ok, err := ks.Next.Has(ctx, k)
	if err != nil {
		span.Error("could not check for presence of key", err)
		return false, err
	}

	span.SetAttributes(
		telemetry.Bool("key_present", ok),
	)

	ks.PairIO.Add(ctx, 1, telemetry.ReadDirection)

	span.Debug("checked for presence of key")

	return ok, nil
}

func (ks *keyspace) Set(ctx context.Context, k, v []byte) error {
	keySize := int64(len(k))
	valueSize := int64(len(v))

	ctx, span := ks.Telemetry.StartSpan(
		ctx,
		"keyspace.set",
		telemetry.If(
			isShortASCII(k),
			telemetry.String("key", string(k)),
		),
		telemetry.Int("key_size", keySize),
		telemetry.If(
			isShortASCII(k),
			telemetry.String("value", string(v)),
		),
		telemetry.Int("value_size", valueSize),
	)
	defer span.End()

	ks.DataIO.Add(ctx, keySize+valueSize, telemetry.WriteDirection)
	ks.PairIO.Add(ctx, 1, telemetry.WriteDirection)
	ks.KeySize.Record(ctx, keySize, telemetry.WriteDirection)
	ks.ValueSize.Record(ctx, valueSize, telemetry.WriteDirection)

	if err := ks.Next.Set(ctx, k, v); err != nil {
		span.Error("could not set key/value pair", err)
		return err
	}

	if len(v) == 0 {
		span.Debug("deleted key/value pair")
	} else {
		span.Debug("set key/value pair")
	}

	return nil
}

func (ks *keyspace) Range(ctx context.Context, fn kv.RangeFunc) error {
	ctx, span := ks.Telemetry.StartSpan(ctx, "keyspace.range")
	defer span.End()

	return ks.instrumentRange(
		ctx,
		span,
		fn,
		ks.Next.Range,
	)
}

func (ks *keyspace) instrumentRange(
	ctx context.Context,
	span *telemetry.Span,
	fn kv.RangeFunc,
	doRange func(context.Context, kv.RangeFunc) error,
) error {
	var (
		count     uint64
		totalSize int64
		brokeLoop bool
	)

	span.Debug("reading key/value pairs")

	err := doRange(
		ctx,
		func(ctx context.Context, k, v []byte) (bool, error) {
			count++

			keySize := int64(len(k))
			valueSize := int64(len(v))
			totalSize += keySize
			totalSize += valueSize

			ks.DataIO.Add(ctx, keySize+valueSize, telemetry.ReadDirection)
			ks.PairIO.Add(ctx, 1, telemetry.ReadDirection)
			ks.KeySize.Record(ctx, keySize, telemetry.ReadDirection)
			ks.ValueSize.Record(ctx, valueSize, telemetry.ReadDirection)

			ok, err := fn(ctx, k, v)
			if ok || err != nil {
				return ok, err
			}

			brokeLoop = true
			return false, nil
		},
	)

	span.SetAttributes(
		telemetry.Int("pairs_read", count),
		telemetry.Int("bytes_read", totalSize),
		telemetry.Bool("reached_end", !brokeLoop && err == nil),
	)

	if err != nil {
		span.Error("could not read key/value pairs", err)
		return err
	}

	span.Debug("completed reading key/value pairs")

	return nil
}

func (ks *keyspace) Close() error {
	ctx, span := ks.Telemetry.StartSpan(context.Background(), "keyspace.close")
	defer span.End()

	if ks.Next == nil {
		span.Warn("keyspace is already closed")
		return nil
	}

	defer func() {
		ks.Next = nil
		ks.OpenCount.Add(ctx, -1)
	}()

	if err := ks.Next.Close(); err != nil {
		span.Error("could not close keyspace", err)
		return err
	}

	span.Debug("closed keyspace")

	return nil
}

// isShortASCII returns true if k is a non-empty ASCII string short enough that
// it may be included as a telemetry attribute.
func isShortASCII(k []byte) bool {
	if len(k) == 0 || len(k) > 128 {
		return false
	}

	for _, octet := range k {
		if octet < ' ' || octet > '~' {
			return false
		}
	}

	return true
}
