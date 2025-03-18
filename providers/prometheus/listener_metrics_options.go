// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package prometheus

import "github.com/prometheus/client_golang/prometheus"

type listenerMetricsConfig struct {
	options options
	// listenerLifetimeHistogram can be nil.
	listenerLifetimeHistogram *prometheus.HistogramVec
}

type ListenerMetricsOption func(*listenerMetricsConfig)

func (c *listenerMetricsConfig) apply(opts []ListenerMetricsOption) {
	for _, o := range opts {
		o(c)
	}
}

func WithListenerOptions(opts ...Option) ListenerMetricsOption {
	return func(o *listenerMetricsConfig) {
		o.options = opts
	}
}

// WithListenerConnectionLifetimeHistogram turns on recording of listener connection lifetime.
// Histogram metrics can be very expensive for Prometheus to retain and query.
func WithListenerConnectionLifetimeHistogram(opts ...HistogramOption) ListenerMetricsOption {
	return func(o *listenerMetricsConfig) {
		o.listenerLifetimeHistogram = prometheus.NewHistogramVec(
			histogramOptions(opts).apply(prometheus.HistogramOpts{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_lifetime_seconds",
				Help:      "Histogram of lifetime (seconds) of a connection from established to closed.",
				Buckets:   prometheus.DefBuckets,
			}),
			[]string{"listener_name"},
		)
	}
}
