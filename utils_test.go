// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package conntrack_test

import (
	"context"
	"sync"

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
	m                         sync.RWMutex
}

func newTestConnectionTracker() *testConnectionTracker {
	return &testConnectionTracker{
		stats: []conntrack.ConnectionStats{},
	}
}

func (t *testConnectionTracker) TagDialerConnection(ctx context.Context, info *conntrack.DialerConnectionTagInfo) context.Context {
	t.m.Lock()
	defer t.m.Unlock()
	t.dialerConnectionTagCtx = ctx
	t.dialerConnectionTagInfo = info
	return ctx
}

func (t *testConnectionTracker) TagListenerConnection(ctx context.Context, info *conntrack.ListenerConnectionTagInfo) context.Context {
	t.m.Lock()
	defer t.m.Unlock()
	t.listenerConnectionTagCtx = ctx
	t.listenerConnectionTagInfo = info
	return ctx
}

func (t *testConnectionTracker) TrackConnection(_ context.Context, stats conntrack.ConnectionStats) {
	t.m.Lock()
	defer t.m.Unlock()
	t.stats = append(t.stats, stats)
}

func (t *testConnectionTracker) GetStats() []conntrack.ConnectionStats {
	t.m.RLock()
	s := t.stats
	t.m.RUnlock()
	return s
}

func (t *testConnectionTracker) GetDialerTagInfo() (context.Context, *conntrack.DialerConnectionTagInfo) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.dialerConnectionTagCtx, t.dialerConnectionTagInfo
}

func (t *testConnectionTracker) GetListenerTagInfo() (context.Context, *conntrack.ListenerConnectionTagInfo) {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.listenerConnectionTagCtx, t.listenerConnectionTagInfo
}
