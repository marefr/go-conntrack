// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package prometheus_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func matchMetricWithLabels(t *testing.T, gatherer prometheus.Gatherer, metricName string, matchingLabels ...*io_prometheus_client.LabelPair) *io_prometheus_client.Metric {
	metricFamilies, err := gatherer.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if mf.GetName() != metricName {
			continue
		}

		for _, m := range mf.GetMetric() {
			found := 0
			for _, lPair := range matchingLabels {
				for _, l := range m.Label {
					if l.GetName() == lPair.GetName() && l.GetValue() == lPair.GetValue() {
						found++
					}
				}
			}

			if found == len(matchingLabels) {
				return m
			}
		}
	}

	return nil
}

func sumCountersForMetricAndLabels(t *testing.T, gatherer prometheus.Gatherer, metricName string, matchingLabels ...*io_prometheus_client.LabelPair) float64 {
	m := matchMetricWithLabels(t, gatherer, metricName, matchingLabels...)
	if m == nil {
		return 0
	}

	if m.Counter != nil {
		return m.GetCounter().GetValue()
	}

	if m.Gauge != nil {
		return m.GetGauge().GetValue()
	}

	return 0
}

func makeLabelPair(name string, value string) *io_prometheus_client.LabelPair {
	return &io_prometheus_client.LabelPair{
		Name:  &name,
		Value: &value,
	}
}
