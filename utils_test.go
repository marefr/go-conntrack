package conntrack_test

import (
	"context"

	"github.com/marefr/go-conntrack/v2"
)

var _ conntrack.ConnectionTracker = &testConnectionTracker{}
var _ conntrack.DialerConnectionTagger = &testConnectionTracker{}
var _ conntrack.ListenerConnectionTagger = &testConnectionTracker{}

type testConnectionTracker struct {
	stats                     []conntrack.ConnectionStats
	dialerConnectionTagCtx    context.Context
	dialerConnectionTagInfo   *conntrack.DialerConnectionTagInfo
	listenerConnectionTagCtx  context.Context
	listenerConnectionTagInfo *conntrack.ListenerConnectionTagInfo
}

func newTestConnectionTracker() *testConnectionTracker {
	return &testConnectionTracker{
		stats: []conntrack.ConnectionStats{},
	}
}

func (t *testConnectionTracker) TagDialerConnection(ctx context.Context, info *conntrack.DialerConnectionTagInfo) context.Context {
	t.dialerConnectionTagCtx = ctx
	t.dialerConnectionTagInfo = info
	return ctx
}

func (t *testConnectionTracker) TagListenerConnection(ctx context.Context, info *conntrack.ListenerConnectionTagInfo) context.Context {
	t.listenerConnectionTagCtx = ctx
	t.listenerConnectionTagInfo = info
	return ctx
}

func (t *testConnectionTracker) TrackConnection(_ context.Context, stats conntrack.ConnectionStats) {
	t.stats = append(t.stats, stats)
}
