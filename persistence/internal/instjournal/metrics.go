package instjournal

import (
	"github.com/dogmatiq/veracity/internal/telemetry"
	"go.opentelemetry.io/otel/metric"
)

type metrics struct {
	OpenCount     metric.Int64UpDownCounter
	ConflictCount metric.Int64Counter
	DataIO        metric.Int64Counter
	RecordIO      metric.Int64Counter
	RecordSize    metric.Int64Histogram
}

func newMetrics(r *telemetry.Recorder) metrics {
	var (
		m   metrics
		err error
	)

	m.OpenCount, err = r.Meter.Int64UpDownCounter(
		"open_journals",
		metric.WithDescription("The number of journals that are currently open."),
		metric.WithUnit("{journal}"),
	)
	if err != nil {
		panic(err)
	}

	m.ConflictCount, err = r.Meter.Int64Counter(
		"conflicts",
		metric.WithDescription("The number of times appending a record to the journal has failed due to a version conflict."),
		metric.WithUnit("{conflict}"),
	)
	if err != nil {
		panic(err)
	}

	m.DataIO, err = r.Meter.Int64Counter(
		"io",
		metric.WithDescription("The cumulative size of the journal records that have been read and written."),
		metric.WithUnit("By"),
	)
	if err != nil {
		panic(err)
	}

	m.RecordIO, err = r.Meter.Int64Counter(
		"record.io",
		metric.WithDescription("The number of journal records being read and written."),
		metric.WithUnit("{record}"),
	)
	if err != nil {
		panic(err)
	}

	m.RecordSize, err = r.Meter.Int64Histogram(
		"record.size",
		metric.WithDescription("The size of journal records that have been read and written."),
		metric.WithUnit("By"),
	)
	if err != nil {
		panic(err)
	}

	return m
}
