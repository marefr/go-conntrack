package conntrack

import (
	"context"
	"net"
	"sync"
)

const DefaultDialerName = "default"

type DialContextFunc func(ctx context.Context, network, addr string) (net.Conn, error)

type DialOptions struct {
	name                  string
	parentDialContextFunc DialContextFunc
	dialObserverFactory   DialObserverFactory
}

type DialOption func(options *DialOptions)

// DialWithName sets the name of the dialer for tracking and monitoring.
// This is the name for the dialer (default is `default`), but for `NewDialContextFunc` can be overwritten from the
// Context using `DialNameToContext`.
func DialWithName(name string) DialOption {
	return func(opts *DialOptions) {
		opts.name = name
	}
}

// DialWithObserver providers an observer to observe the dialer.
func DialWithObserver(observer DialObserverFactory) DialOption {
	return func(opts *DialOptions) {
		opts.dialObserverFactory = observer
	}
}

// DialWithDialer allows you to override the `net.Dialer` instance used to actually conduct the dials.
func DialWithDialer(parentDialer *net.Dialer) DialOption {
	return DialWithDialContextFunc(parentDialer.DialContext)
}

// DialWithDialContextFunc allows you to override func gets used for the actual dialing. The default is `net.Dialer.DialContext`.
func DialWithDialContextFunc(parentDialerFunc DialContextFunc) DialOption {
	return func(opts *DialOptions) {
		opts.parentDialContextFunc = parentDialerFunc
	}
}

// NewDialContextFunc returns a `DialContext` function that tracks outbound connections.
// The signature is compatible with `http.Tranport.DialContext` and is meant to be used there.
func NewDialContextFunc(options ...DialOption) DialContextFunc {
	opts := &DialOptions{name: DefaultDialerName, parentDialContextFunc: (&net.Dialer{}).DialContext}
	for _, opt := range options {
		opt(opts)
	}

	var observer DialObserver
	if opts.dialObserverFactory != nil {
		observer = opts.dialObserverFactory.NewDialObserver(opts.name)
	} else {
		observer = NoOpDialObserver()
	}

	return DialContextFunc(func(ctx context.Context, network, addr string) (net.Conn, error) {
		if ctxName := DialNameFromContext(ctx); ctxName != "" {
			if opts.dialObserverFactory != nil {
				observer = opts.dialObserverFactory.NewDialObserver(ctxName)
			}
		}

		return dialClientConnTracker(ctx, network, addr, observer, opts)
	})
}

type clientConnTracker struct {
	net.Conn
	ctx      context.Context
	opts     *DialOptions
	observer DialObserver
	mu       sync.Mutex
}

func dialClientConnTracker(ctx context.Context, network string, addr string, observer DialObserver, opts *DialOptions) (net.Conn, error) {
	observer.DialAttempt(ctx, addr)

	conn, err := opts.parentDialContextFunc(ctx, network, addr)
	if err != nil {
		observer.DialFailed(ctx, err)
		return nil, err
	}

	observer.ConnectionEstablished(ctx, conn)

	tracker := &clientConnTracker{
		Conn:     conn,
		ctx:      ctx,
		opts:     opts,
		observer: observer,
	}
	return tracker, nil
}

func (ct *clientConnTracker) Close() error {
	err := ct.Conn.Close()
	ct.mu.Lock()
	ct.observer.ConnectionClosed(ct.ctx, err)
	ct.mu.Unlock()

	return err
}

type dialCtxKey struct{}

// DialNameToContext returns a context that will contain a dialer name override.
func DialNameToContext(ctx context.Context, dialerName string) context.Context {
	return context.WithValue(ctx, dialCtxKey{}, dialerName)
}

// DialNameFromContext returns the name of the dialer from the context of the DialContext func, if any.
func DialNameFromContext(ctx context.Context) string {
	val, ok := ctx.Value(dialCtxKey{}).(string)
	if !ok {
		return ""
	}
	return val
}
