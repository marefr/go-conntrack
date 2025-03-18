package conntrack

import (
	"context"
	"net"
	"time"
)

type DialerOptions struct {
	name                  string
	parentDialContextFunc dialerContextFunc
	tracker               ConnectionTracker
}

// DialerOption defines a config option you can set on the dialer.
type DialerOption func(*DialerOptions)

type dialerContextFunc func(context.Context, string, string) (net.Conn, error)

// DialWithName sets the name of the dialer for tracking and monitoring.
// This is the name for the dialer (default is `default`), but for `NewDialContextFunc` can be overwritten from the
// Context using `DialNameToContext`.
func DialWithName(name string) DialerOption {
	return func(opts *DialerOptions) {
		opts.name = name
	}
}

// DialWithTrackers register [ConnectionTracker] trackers that tracks the dialer interactions.
func DialWithTrackers(trackers ...ConnectionTracker) DialerOption {
	return func(opts *DialerOptions) {
		opts.tracker = ChainConnectionTrackers(trackers...)
	}
}

// DialWithDialer allows you to override the `net.Dialer` instance used to actually conduct the dials.
func DialWithDialer(parentDialer *net.Dialer) DialerOption {
	return DialWithDialContextFunc(parentDialer.DialContext)
}

// DialWithDialContextFunc allows you to override func gets used for the actual dialing. The default is `net.Dialer.DialContext`.
func DialWithDialContextFunc(parentDialerFunc dialerContextFunc) DialerOption {
	return func(opts *DialerOptions) {
		opts.parentDialContextFunc = parentDialerFunc
	}
}

type dialerNameKey struct{}

// WithDialName returns a context that will contain a dialer name override.
func WithDialName(ctx context.Context, dialerName string) context.Context {
	return context.WithValue(ctx, dialerNameKey{}, dialerName)
}

// DialNameFromContext returns the name of the dialer from the context of the DialContext func, if any.
func DialNameFromContext(ctx context.Context) string {
	val, ok := ctx.Value(dialerNameKey{}).(string)
	if !ok {
		return ""
	}
	return val
}

// NewDialContextFunc returns a `DialContext` function that tracks outbound connections.
// The signature is compatible with `http.Tranport.DialContext` and is meant to be used there.
func NewDialContextFunc(optFuncs ...DialerOption) func(context.Context, string, string) (net.Conn, error) {
	opts := &DialerOptions{name: defaultName, parentDialContextFunc: (&net.Dialer{}).DialContext}
	for _, f := range optFuncs {
		f(opts)
	}

	return func(ctx context.Context, network string, addr string) (net.Conn, error) {
		name := opts.name
		if ctxName := DialNameFromContext(ctx); ctxName != "" {
			name = ctxName
		} else {
			ctx = WithDialName(ctx, name)
		}
		return dialClientConnTracker(ctx, network, addr, name, opts)
	}
}

func dialClientConnTracker(ctx context.Context, network string, addr string, dialerName string, opts *DialerOptions) (net.Conn, error) {
	ctx = context.WithoutCancel(ctx)
	if t, ok := opts.tracker.(DialerConnectionTagger); ok {
		ctx = t.TagDialerConnection(ctx, &DialerConnectionTagInfo{
			DialerName: dialerName,
			Network:    network,
			Addr:       addr,
		})
	}

	beginTime := time.Now()
	opts.tracker.TrackConnection(ctx, &ConnectionAttempt{
		Client:    true,
		Attempt:   1,
		BeginTime: beginTime,
	})

	conn, err := opts.parentDialContextFunc(ctx, network, addr)
	if err != nil {
		opts.tracker.TrackConnection(ctx, &ConnectionAttemptFailed{
			Client:    true,
			Attempt:   1,
			Error:     err,
			Reason:    FailureReasonFromError(err),
			BeginTime: beginTime,
			EndTime:   time.Now(),
		})

		opts.tracker.TrackConnection(ctx, &ConnectionFailed{
			Client:    true,
			Attempts:  1,
			Error:     err,
			Reason:    FailureReasonFromError(err),
			BeginTime: beginTime,
			EndTime:   time.Now(),
		})

		return nil, err
	}

	establishedTime := time.Now()
	opts.tracker.TrackConnection(ctx, &ConnectionEstablished{
		Client:     true,
		Attempts:   1,
		BeginTime:  beginTime,
		EndTime:    establishedTime,
		LocalAddr:  conn.LocalAddr(),
		RemoteAddr: conn.RemoteAddr(),
	})

	return newConnectionCloseTracker(ctx, conn, true, opts.tracker, establishedTime), nil
}
