package prometheus

import (
	"time"

	"github.com/marefr/go-conntrack/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &ListenerMetrics{}

type ListenerMetrics struct {
	listenerAttemptedTotal    *prometheus.CounterVec
	listenerAcceptedTotal     *prometheus.CounterVec
	listenerConnFailedTotal   *prometheus.CounterVec
	listenerClosedTotal       *prometheus.CounterVec
	listenerOpen              *prometheus.GaugeVec
	listenerLifetimeHistogram *prometheus.HistogramVec
}

// NewListenerMetrics returns a new ListenerMetrics object that has listener tracker methods.
// NOTE: Remember to register ListenerMetrics object by using prometheus registry
// e.g. prometheus.MustRegister(myListenerMetrics).
func NewListenerMetrics(opts ...ListenerMetricsOption) *ListenerMetrics {
	var config listenerMetricsConfig
	config.apply(opts)

	return &ListenerMetrics{
		listenerAttemptedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_attempted_total",
				Help:      "Total number of connections attempted to the given listener of a given name.",
			}), []string{"listener_name"}),
		listenerAcceptedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_accepted_total",
				Help:      "Total number of connections opened to the listener of a given name.",
			}), []string{"listener_name"}),
		listenerConnFailedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_failed_total",
				Help:      "Total number of connections failed to accept to the listener of a given name.",
			}), []string{"listener_name", "reason"}),
		listenerClosedTotal: prometheus.NewCounterVec(
			config.options.applyCounter(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_closed_total",
				Help:      "Total number of connections closed that were made to the listener of a given name.",
			}), []string{"listener_name"}),
		listenerOpen: prometheus.NewGaugeVec(
			config.options.applyGauge(Options{
				Namespace: "net",
				Subsystem: "conntrack",
				Name:      "listener_conn_open",
				Help:      "Number of open connections to the listener of a given name.",
			}), []string{"listener_name"}),
		listenerLifetimeHistogram: config.listenerLifetimeHistogram,
	}
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (m *ListenerMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.listenerAttemptedTotal.Describe(ch)
	m.listenerAcceptedTotal.Describe(ch)
	m.listenerConnFailedTotal.Describe(ch)
	m.listenerClosedTotal.Describe(ch)
	m.listenerOpen.Describe(ch)

	if m.listenerLifetimeHistogram != nil {
		m.listenerLifetimeHistogram.Describe(ch)
	}
}

// Collect is called by the Prometheus registry when collecting
// metrics. The implementation sends each collected metric via the
// provided channel and returns once the last metric has been sent.
func (m *ListenerMetrics) Collect(ch chan<- prometheus.Metric) {
	m.listenerAttemptedTotal.Collect(ch)
	m.listenerAcceptedTotal.Collect(ch)
	m.listenerConnFailedTotal.Collect(ch)
	m.listenerClosedTotal.Collect(ch)
	m.listenerOpen.Collect(ch)

	if m.listenerLifetimeHistogram != nil {
		m.listenerLifetimeHistogram.Collect(ch)
	}
}

// InitializeMetrics initializes all metrics, with their appropriate null
// value, for all known listener names. This is useful, to
// ensure that all metrics exist when collecting and querying.
func (m *ListenerMetrics) InitializeMetrics(listenerNames ...string) {
	for _, name := range listenerNames {
		// These are just references (no increments), as just referencing will create the labels but not set values.
		_, _ = m.listenerAttemptedTotal.GetMetricWithLabelValues(name)
		_, _ = m.listenerAcceptedTotal.GetMetricWithLabelValues(name)
		_, _ = m.listenerConnFailedTotal.GetMetricWithLabelValues(name, conntrack.FailureReasonUnknown.String())
		_, _ = m.listenerClosedTotal.GetMetricWithLabelValues(name)
		_, _ = m.listenerOpen.GetMetricWithLabelValues(name)

		if m.listenerLifetimeHistogram != nil {
			_, _ = m.listenerLifetimeHistogram.GetMetricWithLabelValues(name)
		}
	}
}

func (m *ListenerMetrics) TrackListener() conntrack.ConnectionTracker {
	return newListenerReporter(m)
}

func (m *ListenerMetrics) reportListenerConnAttempt(listenerName string) {
	m.listenerAttemptedTotal.WithLabelValues(listenerName).Inc()
}

func (m *ListenerMetrics) reportListenerConnAccepted(listenerName string) {
	m.listenerAcceptedTotal.WithLabelValues(listenerName).Inc()
	m.listenerOpen.WithLabelValues(listenerName).Inc()
}

func (m *ListenerMetrics) reportListenerConnFailed(listenerName string, failureReason conntrack.FailureReason) {
	m.listenerConnFailedTotal.WithLabelValues(listenerName, failureReason.String()).Inc()
}

func (m *ListenerMetrics) reportListenerConnClosed(listenerName string, lifetime time.Duration) {
	m.listenerClosedTotal.WithLabelValues(listenerName).Inc()
	m.listenerOpen.WithLabelValues(listenerName).Dec()

	if m.listenerLifetimeHistogram != nil {
		m.listenerLifetimeHistogram.WithLabelValues(listenerName).Observe(lifetime.Seconds())
	}
}
