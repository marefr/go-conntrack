package prometheus

import (
	"time"

	"github.com/marefr/go-conntrack/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &DialerMetrics{}

type DialerMetrics struct {
	dialerAttemptedTotal       *prometheus.CounterVec
	dialerConnEstablishedTotal *prometheus.CounterVec
	dialerConnFailedTotal      *prometheus.CounterVec
	dialerConnClosedTotal      *prometheus.CounterVec
	dialerConnOpen             *prometheus.GaugeVec
	dialerLifetimeHistogram    *prometheus.HistogramVec
}

// NewDialerMetrics returns a new DialerMetrics object that has dialer tracker methods.
// NOTE: Remember to register DialerMetrics object by using prometheus registry
// e.g. prometheus.MustRegister(myDialerMetrics).
func NewDialerMetrics(opts ...ClientMetricsOption) *DialerMetrics {
	var config dialerMetricsConfig
	config.apply(opts)

	return &DialerMetrics{
		dialerAttemptedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_attempted_total",
				Help:      "Total number of connections attempted by the given dialer of a given name.",
			}), []string{"dialer_name"}),
		dialerConnEstablishedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_established_total",
				Help:      "Total number of connections successfully established by the given dialer of a given name.",
			}), []string{"dialer_name"}),
		dialerConnFailedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_failed_total",
				Help:      "Total number of connections failed to dial by the dialer of a given name.",
			}), []string{"dialer_name", "reason"}),
		dialerConnClosedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_closed_total",
				Help:      "Total number of connections closed which originated from the dialer of a given name.",
			}), []string{"dialer_name"}),
		dialerConnOpen: prometheus.NewGaugeVec(
			config.options.applyGauge(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "dialer_conn_open",
				Help:      "Number of open connections which originated from the dialer of a given name.",
			}), []string{"dialer_name"}),
		dialerLifetimeHistogram: config.dialerLifetimeHistogram,
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *DialerMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.dialerAttemptedTotal.Describe(ch)
	m.dialerConnEstablishedTotal.Describe(ch)
	m.dialerConnFailedTotal.Describe(ch)
	m.dialerConnClosedTotal.Describe(ch)
	m.dialerConnOpen.Describe(ch)

	if m.dialerLifetimeHistogram != nil {
		m.dialerLifetimeHistogram.Describe(ch)
	}
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *DialerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.dialerAttemptedTotal.Collect(ch)
	m.dialerConnEstablishedTotal.Collect(ch)
	m.dialerConnFailedTotal.Collect(ch)
	m.dialerConnClosedTotal.Collect(ch)
	m.dialerConnOpen.Collect(ch)

	if m.dialerLifetimeHistogram != nil {
		m.dialerLifetimeHistogram.Collect(ch)
	}
}

// InitializeMetrics initializes all metrics, with their appropriate null
// value, for all known dialer names. This is useful, to
// ensure that all metrics exist when collecting and querying.
func (m *DialerMetrics) InitializeMetrics(dialerNames ...string) {
	for _, name := range dialerNames {
		// These are just references (no increments), as just referencing will create the labels but not set values.
		_, _ = m.dialerAttemptedTotal.GetMetricWithLabelValues(name)
		_, _ = m.dialerConnEstablishedTotal.GetMetricWithLabelValues(name)

		for _, reason := range conntrack.FailureReasons() {
			_, _ = m.dialerConnFailedTotal.GetMetricWithLabelValues(name, reason.String())
		}

		_, _ = m.dialerConnClosedTotal.GetMetricWithLabelValues(name)
		_, _ = m.dialerConnOpen.GetMetricWithLabelValues(name)

		if m.dialerLifetimeHistogram != nil {
			_, _ = m.dialerLifetimeHistogram.GetMetricWithLabelValues(name)
		}
	}
}

func (m *DialerMetrics) TrackDialer() conntrack.ConnectionTracker {
	return newDialerReporter(m)
}

func (m *DialerMetrics) reportDialerConnAttempt(dialerName string) {
	m.dialerAttemptedTotal.WithLabelValues(dialerName).Inc()
}

func (m *DialerMetrics) reportDialerConnEstablished(dialerName string) {
	m.dialerConnEstablishedTotal.WithLabelValues(dialerName).Inc()
	m.dialerConnOpen.WithLabelValues(dialerName).Inc()
}

func (m *DialerMetrics) reportDialerConnClosed(dialerName string, lifetime time.Duration) {
	m.dialerConnClosedTotal.WithLabelValues(dialerName).Inc()
	m.dialerConnOpen.WithLabelValues(dialerName).Dec()

	if m.dialerLifetimeHistogram != nil {
		m.dialerLifetimeHistogram.WithLabelValues(dialerName).Observe(lifetime.Seconds())
	}
}

func (m *DialerMetrics) reportDialerConnFailed(dialerName string, failureReason conntrack.FailureReason) {
	m.dialerConnFailedTotal.WithLabelValues(dialerName, failureReason.String()).Inc()
}
