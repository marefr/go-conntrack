// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package prometheus

import "github.com/prometheus/client_golang/prometheus"

type dialerMetricsConfig struct {
	options options
	// dialerLifetimeHistogram can be nil.
	dialerLifetimeHistogram *prometheus.HistogramVec
}

type ClientMetricsOption func(*dialerMetricsConfig)

func (c *dialerMetricsConfig) apply(opts []ClientMetricsOption) {
	for _, o := range opts {
		o(c)
	}
}

func WithDialerOptions(opts ...Option) ClientMetricsOption {
	return func(o *dialerMetricsConfig) {
		o.options = opts
	}
}

// WithDialerConnectionLifetimeHistogram turns on recording of dialer connection lifetime.
// Histogram metrics can be very expensive for Prometheus to retain and query.
func WithDialerConnectionLifetimeHistogram(opts ...HistogramOption) ClientMetricsOption {
	return func(o *dialerMetricsConfig) {
		o.dialerLifetimeHistogram = prometheus.NewHistogramVec(
			histogramOptions(opts).apply(prometheus.HistogramOpts{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_lifetime_seconds",
				Help:      "Histogram of lifetime (seconds) of a connection from established to closed.",
				Buckets:   prometheus.DefBuckets,
			}),
			[]string{"dialer_name"},
		)
	}
}
