package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// A Option lets you add options to metrics using With* funcs.
type Option func(*Options)

type Options struct {
	// Namespace, Subsystem, and Name are components of the fully-qualified
	// name of the Metric (created by joining these components with
	// "_"). Only Name is mandatory, the others merely help structuring the
	// name. Note that the fully-qualified name of the metric must be a
	// valid Prometheus metric name.
	Namespace string
	Subsystem string
	Name      string

	// Help provides information about this metric.
	//
	// Metrics with the same fully-qualified name must have the same Help
	// string.
	Help string

	// ConstLabels are used to attach fixed labels to this metric. Metrics
	// with the same fully-qualified name must have the same label names in
	// their ConstLabels.
	//
	// ConstLabels are only used rarely. In particular, do not use them to
	// attach the same labels to all your metrics. Those use cases are
	// better covered by target labels set by the scraping Prometheus
	// server, or by one specific metric (e.g. a build_info or a
	// machine_role metric). See also
	// https://prometheus.io/docs/instrumenting/writing_exporters/#target-labels-not-static-scraped-labels
	ConstLabels prometheus.Labels
}

type options []Option

func (co options) applyCounter(o Options) prometheus.CounterOpts {
	for _, f := range co {
		f(&o)
	}

	return prometheus.CounterOpts{
		Namespace:   o.Namespace,
		Subsystem:   o.Subsystem,
		Name:        o.Name,
		Help:        o.Help,
		ConstLabels: o.ConstLabels,
	}
}

func (co options) applyGauge(o Options) prometheus.GaugeOpts {
	for _, f := range co {
		f(&o)
	}

	return prometheus.GaugeOpts{
		Namespace:   o.Namespace,
		Subsystem:   o.Subsystem,
		Name:        o.Name,
		Help:        o.Help,
		ConstLabels: o.ConstLabels,
	}
}

// WithConstLabels allows you to add ConstLabels to metrics.
func WithConstLabels(labels prometheus.Labels) Option {
	return func(o *Options) {
		o.ConstLabels = labels
	}
}

// WithSubsystem allows you to add a Subsystem to metrics.
func WithSubsystem(subsystem string) Option {
	return func(o *Options) {
		o.Subsystem = subsystem
	}
}

// WithNamespace allows you to add a Namespace to metrics.
func WithNamespace(namespace string) Option {
	return func(o *Options) {
		o.Namespace = namespace
	}
}

// A HistogramOption lets you add options to Histogram metrics using With*
// funcs.
type HistogramOption func(*prometheus.HistogramOpts)

type histogramOptions []HistogramOption

func (ho histogramOptions) apply(o prometheus.HistogramOpts) prometheus.HistogramOpts {
	for _, f := range ho {
		f(&o)
	}
	return o
}

// WithHistogramBuckets allows you to specify custom bucket ranges for histograms if EnableHandlingTimeHistogram is on.
func WithHistogramBuckets(buckets []float64) HistogramOption {
	return func(o *prometheus.HistogramOpts) { o.Buckets = buckets }
}

// WithHistogramOpts allows you to specify HistogramOpts but makes sure the correct name and label is used.
// This function is helpful when specifying more than just the buckets, like using NativeHistograms.
func WithHistogramOpts(opts *prometheus.HistogramOpts) HistogramOption {
	// TODO: This isn't ideal either if new fields are added to prometheus.HistogramOpts.
	// Maybe we can change the interface to accept arbitrary HistogramOpts and
	// only make sure to overwrite the necessary fields (name, labels).
	return func(o *prometheus.HistogramOpts) {
		o.Buckets = opts.Buckets
		o.NativeHistogramBucketFactor = opts.NativeHistogramBucketFactor
		o.NativeHistogramZeroThreshold = opts.NativeHistogramZeroThreshold
		o.NativeHistogramMaxBucketNumber = opts.NativeHistogramMaxBucketNumber
		o.NativeHistogramMinResetDuration = opts.NativeHistogramMinResetDuration
		o.NativeHistogramMaxZeroThreshold = opts.NativeHistogramMaxZeroThreshold
	}
}

// WithHistogramConstLabels allows you to add custom ConstLabels to
// histograms metrics.
func WithHistogramConstLabels(labels prometheus.Labels) HistogramOption {
	return func(o *prometheus.HistogramOpts) {
		o.ConstLabels = labels
	}
}

// WithHistogramSubsystem allows you to add a Subsystem to histograms metrics.
func WithHistogramSubsystem(subsystem string) HistogramOption {
	return func(o *prometheus.HistogramOpts) {
		o.Subsystem = subsystem
	}
}

// WithHistogramNamespace allows you to add a Namespace to histograms metrics.
func WithHistogramNamespace(namespace string) HistogramOption {
	return func(o *prometheus.HistogramOpts) {
		o.Namespace = namespace
	}
}
