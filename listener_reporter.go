// Copyright 2016 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package conntrack

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	listenerAcceptedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "net",
			Subsystem: "conntrack",
			Name:      "listener_conn_accepted_total",
			Help:      "Total number of connections opened to the listener of a given name.",
		}, []string{"listener_name"})

	listenerClosedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "net",
			Subsystem: "conntrack",
			Name:      "listener_conn_closed_total",
			Help:      "Total number of connections closed that were made to the listener of a given name.",
		}, []string{"listener_name"})
	listenerOpen = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "net",
			Subsystem: "conntrack",
			Name:      "listener_conn_open",
			Help:      "Number of open connections to the listener of a given name.",
		}, []string{"listener_name"})
)

// preRegisterListener pre-populates Prometheus labels for the given listener name, to avoid Prometheus missing labels issue.
func preRegisterListenerMetrics(listenerName string) {
	listenerAcceptedTotal.WithLabelValues(listenerName)
	listenerClosedTotal.WithLabelValues(listenerName)
	listenerOpen.WithLabelValues(listenerName)
}

func reportListenerConnAccepted(listenerName string) {
	listenerAcceptedTotal.WithLabelValues(listenerName).Inc()
	listenerOpen.WithLabelValues(listenerName).Inc()
}

func reportListenerConnClosed(listenerName string) {
	listenerClosedTotal.WithLabelValues(listenerName).Inc()
	listenerOpen.WithLabelValues(listenerName).Dec()
}
