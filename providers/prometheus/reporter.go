// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package prometheus

import (
	"context"

	"github.com/marefr/go-conntrack/v2"
)

type reporter struct {
	dialerMetrics   *DialerMetrics
	listenerMetrics *ListenerMetrics
}

func newDialerReporter(dialerMetrics *DialerMetrics) *reporter {
	return &reporter{
		dialerMetrics: dialerMetrics,
	}
}

func newListenerReporter(listenerMetrics *ListenerMetrics) *reporter {
	return &reporter{
		listenerMetrics: listenerMetrics,
	}
}

func (r reporter) TrackConnection(ctx context.Context, stats conntrack.ConnectionStats) {
	if stats.IsClient() {
		r.trackDialerConnection(ctx, stats)
		return
	}

	r.trackListenerConnection(ctx, stats)
}

func (r reporter) trackDialerConnection(ctx context.Context, stats conntrack.ConnectionStats) {
	dialerName := conntrack.DialNameFromContext(ctx)

	switch s := stats.(type) {
	case *conntrack.ConnectionAttempt:
		r.dialerMetrics.reportDialerConnAttempt(dialerName)
	case *conntrack.ConnectionFailed:
		r.dialerMetrics.reportDialerConnFailed(dialerName, s.Reason)
	case *conntrack.ConnectionEstablished:
		r.dialerMetrics.reportDialerConnEstablished(dialerName)
	case *conntrack.ConnectionClosed:
		r.dialerMetrics.reportDialerConnClosed(dialerName, s.EndTime.Sub(s.BeginTime))
	}
}

func (r reporter) trackListenerConnection(ctx context.Context, stats conntrack.ConnectionStats) {
	listenerName := conntrack.ListenerNameFromContext(ctx)

	switch s := stats.(type) {
	case *conntrack.ConnectionAttempt:
		r.listenerMetrics.reportListenerConnAttempt(listenerName)
	case *conntrack.ConnectionFailed:
		r.listenerMetrics.reportListenerConnFailed(listenerName, s.Reason)
	case *conntrack.ConnectionEstablished:
		r.listenerMetrics.reportListenerConnAccepted(listenerName)
	case *conntrack.ConnectionClosed:
		r.listenerMetrics.reportListenerConnClosed(listenerName, s.EndTime.Sub(s.BeginTime))
	}
}
