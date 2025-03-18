package trace

import (
	"context"
	"fmt"

	"github.com/marefr/go-conntrack/v2"
	"golang.org/x/net/trace"
)

var _ conntrack.DialerConnectionTagger = &tracer{}
var _ conntrack.ListenerConnectionTagger = &tracer{}

type tracer struct{}

// New returns a new [conntrack.ConnectionTracker] that traces dialer (client) and listener (server) connection interactions.
func New() conntrack.ConnectionTracker {
	return &tracer{}
}

func (t tracer) TagDialerConnection(ctx context.Context, info *conntrack.DialerConnectionTagInfo) context.Context {
	return withTraceCtx(ctx, &traceCtx{
		name:     info.DialerName,
		eventLog: trace.NewEventLog(fmt.Sprintf("net.ClientConn.%s", info.DialerName), info.Addr),
	})
}

func (t tracer) TagListenerConnection(ctx context.Context, info *conntrack.ListenerConnectionTagInfo) context.Context {
	return withTraceCtx(ctx, &traceCtx{
		name: info.ListenerName,
	})
}

func (t tracer) TrackConnection(ctx context.Context, stats conntrack.ConnectionStats) {
	traceCtx := traceCtxFromContext(ctx)
	if traceCtx == nil {
		return
	}

	if stats.IsClient() {
		trackDialerConnection(stats, traceCtx)
		return
	}

	trackListenerConnection(stats, traceCtx)
}

func trackDialerConnection(stats conntrack.ConnectionStats, traceCtx *traceCtx) {
	switch s := stats.(type) {
	case *conntrack.ConnectionEstablished:
		if traceCtx.eventLog == nil {
			return
		}

		traceCtx.eventLog.Printf("accepted: %s -> %s", s.RemoteAddr, s.LocalAddr)
	case *conntrack.ConnectionClosed:
		if traceCtx.eventLog == nil {
			return
		}

		if s.Error != nil {
			traceCtx.eventLog.Errorf("failed closing: %v", s.Error)
		} else {
			traceCtx.eventLog.Printf("closing")
		}

		traceCtx.eventLog.Finish()
		traceCtx.eventLog = nil
	}
}

func trackListenerConnection(stats conntrack.ConnectionStats, traceCtx *traceCtx) {
	switch s := stats.(type) {
	case *conntrack.ConnectionEstablished:
		traceCtx.eventLog = trace.NewEventLog(fmt.Sprintf("net.ServerConn.%s", traceCtx.name), s.RemoteAddr.String())
		traceCtx.eventLog.Printf("accepted: %s -> %s", s.RemoteAddr, s.LocalAddr)
	case *conntrack.ConnectionClosed:
		if traceCtx.eventLog == nil {
			return
		}

		if s.Error != nil {
			traceCtx.eventLog.Errorf("failed closing: %v", s.Error)
		} else {
			traceCtx.eventLog.Printf("closing")
		}

		traceCtx.eventLog.Finish()
		traceCtx.eventLog = nil
	}
}

type traceCtx struct {
	name     string
	eventLog trace.EventLog
}

type traceCtxKey struct{}

func withTraceCtx(ctx context.Context, traceCtx *traceCtx) context.Context {
	return context.WithValue(ctx, traceCtxKey{}, traceCtx)
}

func traceCtxFromContext(ctx context.Context) *traceCtx {
	if val := ctx.Value(traceCtxKey{}); val != nil {
		return val.(*traceCtx)
	}

	return nil
}
