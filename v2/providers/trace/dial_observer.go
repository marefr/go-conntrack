package trace

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/net/trace"

	"github.com/marefr/go-conntrack/v2"
)

// DialObserver creates a new [goconntrack.DialObserverFactory] that observes and trace dialer connections.
func DialObserver() conntrack.DialObserverFactory {
	return conntrack.DialObserverFactoryFunc(func(name string) conntrack.DialObserver {
		return &traceObserver{
			name: name,
		}
	})
}

type traceObserver struct {
	name  string
	event trace.EventLog
}

func (o *traceObserver) DialAttempt(ctx context.Context, addr string) {
	o.event = trace.NewEventLog(fmt.Sprintf("net.ClientConn.%s", o.name), fmt.Sprintf("%v", addr))
}

func (o *traceObserver) DialFailed(ctx context.Context, err error) {
	if o.event == nil {
		return
	}

	o.event.Errorf("failed dialing: %v", err)
	o.event.Finish()
}

func (o *traceObserver) ConnectionEstablished(ctx context.Context, conn net.Conn) {
	if o.event == nil {
		return
	}

	o.event.Printf("established: %s -> %s", conn.LocalAddr(), conn.RemoteAddr())
}

func (o *traceObserver) ConnectionClosed(ctx context.Context, err error) {
	if o.event == nil {
		return
	}

	if err != nil {
		o.event.Errorf("failed closing: %v", err)
	} else {
		o.event.Printf("closing")
	}

	o.event.Finish()
	o.event = nil
}
