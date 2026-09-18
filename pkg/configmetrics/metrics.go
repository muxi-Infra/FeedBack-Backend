package configmetrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	Refresh           *prometheus.CounterVec
	Reconciliations   *prometheus.CounterVec
	Duration          *prometheus.HistogramVec
	ApplyAge          prometheus.Histogram
	InvalidationDelay prometheus.Histogram
	Cache             *prometheus.CounterVec
	Events            *prometheus.CounterVec
	Publish           *prometheus.CounterVec
	Consumer          prometheus.Gauge
	LastFull          prometheus.Gauge
	Pending           prometheus.Gauge
	Unread            prometheus.Gauge
	UnreadExact       prometheus.Gauge
	StreamStatsUp     prometheus.Gauge
	OldestPending     prometheus.Gauge
	OutboxPending     prometheus.Gauge
	OutboxAge         prometheus.Gauge
	OutboxStatsUp     prometheus.Gauge
}

func New(reg *prometheus.Registry) *Metrics {
	const prefix = "feedback_v3_config_"
	counter := func(name string, labels ...string) *prometheus.CounterVec {
		return prometheus.NewCounterVec(prometheus.CounterOpts{Name: prefix + name, Help: name}, labels)
	}
	gauge := func(name string) prometheus.Gauge {
		return prometheus.NewGauge(prometheus.GaugeOpts{Name: prefix + name, Help: name})
	}
	hist := func(name string) prometheus.Histogram {
		return prometheus.NewHistogram(prometheus.HistogramOpts{Name: prefix + name, Help: name, Buckets: []float64{.01, .1, .5, 1, 2, 5, 10, 30, 60, 120, 300}})
	}
	m := &Metrics{
		Reconciliations: counter("reconcile_total", "result"),
		Refresh:         counter("refresh_total", "trigger", "result"), Cache: counter("cache_requests_total", "result"),
		Events: counter("events_total", "stage"), Publish: counter("publish_total", "result"),
		Duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: prefix + "refresh_duration_seconds", Help: "Refresh duration", Buckets: prometheus.DefBuckets}, []string{"trigger"}),
		ApplyAge: hist("apply_age_seconds"), InvalidationDelay: hist("invalidation_delay_seconds"),
		Consumer: gauge("consumer_up"), LastFull: gauge("last_full_success_timestamp_seconds"),
		Pending: gauge("stream_pending"), Unread: gauge("stream_unread"), UnreadExact: gauge("stream_unread_exact"),
		StreamStatsUp: gauge("stream_stats_up"), OldestPending: gauge("oldest_pending_age_seconds"),
		OutboxPending: gauge("outbox_pending"), OutboxAge: gauge("outbox_oldest_age_seconds"),
		OutboxStatsUp: gauge("outbox_stats_up"),
	}
	reg.MustRegister(m.Refresh, m.Reconciliations, m.Duration, m.ApplyAge, m.InvalidationDelay, m.Cache, m.Events, m.Publish, m.Consumer, m.LastFull, m.Pending, m.Unread, m.UnreadExact, m.StreamStatsUp, m.OldestPending, m.OutboxPending, m.OutboxAge, m.OutboxStatsUp)
	return m
}
