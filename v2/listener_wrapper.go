package conntrack

import (
	"net"
	"sync"
	"time"

	"github.com/jpillora/backoff"
)

const DefaultListenerName = "default"

type ListenerOptions struct {
	name            string
	tcpKeepAlive    time.Duration
	retryBackoff    *backoff.Backoff
	observerFactory ListenerObserverFactory
}

type ListenerOption func(options *ListenerOptions)

// ListenerWithName sets the name of the Listener for use in reporting.
func ListenerWithName(name string) ListenerOption {
	return func(l *ListenerOptions) {
		l.name = name
	}
}

// ListenerWithRetries enables retrying of temporary Accept() errors, with the given backoff between attempts.
// Concurrent accept calls that receive temporary errors have independent backoff scaling.
func ListenerWithRetries(b backoff.Backoff) ListenerOption {
	return func(l *ListenerOptions) {
		l.retryBackoff = &b
	}
}

// ListenerWithTCPKeepAlive makes sure that any `net.TCPConn` that get accepted have a keep-alive.
// This is useful for HTTP servers in order for, for example laptops, to not use up resources on the
// server while they don't utilise their connection.
// A value of 0 disables it.
func ListenerWithTCPKeepAlive(keepalive time.Duration) ListenerOption {
	return func(l *ListenerOptions) {
		l.tcpKeepAlive = keepalive
	}
}

// ListenerWithObserver providers an observer to observe the listener.
func ListenerWithObserver(observer ListenerObserverFactory) ListenerOption {
	return func(l *ListenerOptions) {
		l.observerFactory = observer
	}
}

type observedListener struct {
	net.Listener
	opts     *ListenerOptions
	observer ListenerObserver
}

// NewListener returns the given listener wrapped in connection tracking listener.
func NewListener(inner net.Listener, options ...ListenerOption) net.Listener {
	l := observedListener{
		Listener: inner,
		opts: &ListenerOptions{
			name: DefaultListenerName,
		},
	}

	for _, opt := range options {
		opt(l.opts)
	}

	if l.opts.observerFactory != nil {
		l.observer = l.opts.observerFactory.NewListenerObserver(l.opts.name)
	} else {
		l.observer = NoOpListenerObserver()
	}

	return &l
}

func (l *observedListener) Accept() (net.Conn, error) {
	var (
		conn net.Conn
		err  error
	)
	for attempt := 0; ; attempt++ {
		l.observer.AcceptAttempt(attempt)
		conn, err = l.Listener.Accept()
		if err == nil || l.opts.retryBackoff == nil {
			break
		}
		if t, ok := err.(interface{ Temporary() bool }); !ok || !t.Temporary() {
			break
		}
		time.Sleep(l.opts.retryBackoff.ForAttempt(float64(attempt)))
	}
	if err != nil {
		l.observer.AcceptFailed(err)
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok && l.opts.tcpKeepAlive > 0 {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(l.opts.tcpKeepAlive)
	}

	return newObservedConnection(conn, l.observer), nil
}

type observedConnection struct {
	net.Conn
	observer ListenerObserver
	mu       sync.Mutex
}

func newObservedConnection(inner net.Conn, observer ListenerObserver) net.Conn {
	oc := &observedConnection{
		Conn:     inner,
		observer: observer,
	}

	observer.ConnectionAccepted(inner)

	return oc
}

func (oc *observedConnection) Close() error {
	err := oc.Conn.Close()
	oc.mu.Lock()
	oc.observer.ConnectionClosed(err)
	oc.mu.Unlock()
	return err
}
