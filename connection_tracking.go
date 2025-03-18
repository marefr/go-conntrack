package conntrack

import (
	"context"
	"net"
	"os"
	"syscall"
	"time"
)

// DialerConnectionTagInfo defines the relevant information needed by dialer connection context tagger.
type DialerConnectionTagInfo struct {
	// DialerName is the name of the dialer.
	DialerName string

	// RemoteAddr is the remote address of the corresponding connection.
	Network string

	// LocalAddr is the local address of the corresponding connection.
	Addr string
}

// DialerConnectionTagger the interface for tagging (setup) dialer connection contexts.
type DialerConnectionTagger interface {
	// TagDialerConnection
	// Note: The ctx is propagated/derived from outgoing connections.
	TagDialerConnection(ctx context.Context, info *DialerConnectionTagInfo) context.Context
}

// ListenerConnectionTagInfo defines the relevant information needed by listener connection context tagger.
type ListenerConnectionTagInfo struct {
	// ListenerName is the name of the listener.
	ListenerName string
}

// ListenerConnectionTagger the interface for tagging (setup) listener connection contexts.
type ListenerConnectionTagger interface {
	// TagListenerConnection
	// Note: The ctx is not propagated/derived from incoming connections.
	TagListenerConnection(ctx context.Context, info *ListenerConnectionTagInfo) context.Context
}

// ConnectionStats defines the interface used for tracking dialer and listener connection events.
type ConnectionStats interface {
	// IsClient returns true if a client connection triggered the event.
	IsClient() bool
}

// ConnectionAttempt connection tracker event used when a connection attempt is made.
type ConnectionAttempt struct {
	// Client returns true if a client connection triggered the event.
	Client bool

	// Attempt the sequence number of the connection attempt.
	Attempt int

	// BeginTime the tim when the connection attempt started.
	BeginTime time.Time
}

// IsClient returns true if a client connection triggered the event.
func (c ConnectionAttempt) IsClient() bool {
	return c.Client
}

// ConnectionAttemptFailed connection tracker event used when a connection attempt fails.
type ConnectionAttemptFailed struct {
	// Client returns true if a client connection triggered the event.
	Client bool

	// Attempt the sequence number of the connection attempt.
	Attempt int

	// BeginTime the time when the connection attempt started.
	BeginTime time.Time

	// EndTime the time when the connection attempt failed.
	EndTime time.Time

	// Error the error returned when the connection attempt failed.
	Error error

	// Reason the [FailureReason] when the connection attempt failed.
	Reason FailureReason
}

// IsClient returns true if a client connection triggered the event.
func (c ConnectionAttemptFailed) IsClient() bool {
	return c.Client
}

// ConnectionFailed connection tracker event used when a connection fails (after X attempt(s) has failed).
type ConnectionFailed struct {
	// Client returns true if a client connection triggered the event.
	Client bool

	// Attempts the total number of attempts made before the connection failed.
	Attempts int

	// BeginTime the time when the first connection attempt started.
	BeginTime time.Time

	// EndTime the time when the connection failed.
	EndTime time.Time

	// Error the error returned when the connection failed.
	Error error

	// Reason the [FailureReason] when the connection failed.
	Reason FailureReason
}

// IsClient returns true if a client connection triggered the event.
func (c ConnectionFailed) IsClient() bool {
	return c.Client
}

// ConnectionEstablished connection tracker event used when a connection is established.
type ConnectionEstablished struct {
	// Client returns true if a client connection triggered the event.
	Client bool

	// Attempts the total number of attempts made before the connection was established.
	Attempts int

	// BeginTime the time when the first connection attempt started.
	BeginTime time.Time

	// EndTime the time when the connection was established.
	EndTime time.Time

	// LocalAddr the local address of the established connection.
	LocalAddr net.Addr

	// LocalAddr the remote address of the established connection.
	RemoteAddr net.Addr
}

// IsClient returns true if a client connection triggered the event.
func (c ConnectionEstablished) IsClient() bool {
	return c.Client
}

// ConnectionClosed connection tracker event used when a connection is closed.
type ConnectionClosed struct {
	// Client returns true if a client connection triggered the event.
	Client bool

	// BeginTime the time when the connection was established.
	BeginTime time.Time

	// EndTime the time when the connection was closed.
	EndTime time.Time

	// Error the error returned when the connection was closed, if any.
	Error error
}

// IsClient returns true if a client connection triggered the event.
func (c ConnectionClosed) IsClient() bool {
	return c.Client
}

// ConnectionTracker the interface for tracking dialer and listener connections.
type ConnectionTracker interface {
	// TrackConnection tracks the
	TrackConnection(ctx context.Context, stats ConnectionStats)
}

// ConnectionTrackerFunc is an adapter to allow the use of
// ordinary functions as [ConnectionTracker]. If f is a function
// with the appropriate signature, ConnectionTrackerFunc(f) is a
// [ConnectionTracker] that calls f.
type ConnectionTrackerFunc func(ctx context.Context, stats ConnectionStats)

// TrackConnection calls fn(ctx, stats).
func (fn ConnectionTrackerFunc) TrackConnection(ctx context.Context, stats ConnectionStats) {
	fn(ctx, stats)
}

// FailureReason enum defines reasons for connection failures.
type FailureReason string

const (
	// FailureReasonResolution failed to resolve connection address.
	FailureReasonResolution FailureReason = "resolution"

	// FailureReasonConnectionRefused connection was refused.
	FailureReasonConnectionRefused FailureReason = "refused"

	// FailureReasonTimeout connection timed out.
	FailureReasonTimeout FailureReason = "timeout"

	// FailureReasonUnknown unknown reason.
	FailureReasonUnknown FailureReason = "unknown"
)

// String returns [FailureReason] as string. If empty, [FailureReasonUnknown] is returned.
func (fr FailureReason) String() string {
	if string(fr) == "" {
		return string(FailureReasonUnknown)
	}

	return string(fr)
}

// FailureReasons returns all defined failure reasons.
func FailureReasons() []FailureReason {
	return []FailureReason{
		FailureReasonTimeout,
		FailureReasonResolution,
		FailureReasonConnectionRefused,
		FailureReasonUnknown,
	}
}

// FailureReasonFromError extracts the [FailureReason] from the provided error.
func FailureReasonFromError(err error) FailureReason {
	if netErr, ok := err.(*net.OpError); ok {
		switch nestErr := netErr.Err.(type) {
		case *net.DNSError:
			return FailureReasonResolution
		case *os.SyscallError:
			if nestErr.Err == syscall.ECONNREFUSED {
				return FailureReasonConnectionRefused
			}
			return FailureReasonUnknown
		}
		if netErr.Timeout() {
			return FailureReasonTimeout
		}
	} else if err == context.Canceled || err == context.DeadlineExceeded {
		return FailureReasonTimeout
	}

	return FailureReasonUnknown
}

type chainConnectionTracker struct {
	trackers []ConnectionTracker
}

// ChainConnectionTrackers chain multiple [ConnectionTracker] trackers together into a single [ConnectionTracker].
//
// Note: If trackers is nil a nil [ConnectionTracker] will be returned.
func ChainConnectionTrackers(trackers ...ConnectionTracker) ConnectionTracker {
	if trackers == nil {
		return nil
	}

	if len(trackers) == 1 {
		return trackers[0]
	}

	return &chainConnectionTracker{
		trackers: trackers,
	}
}

func (ct chainConnectionTracker) TagDialerConnection(ctx context.Context, info *DialerConnectionTagInfo) context.Context {
	for _, t := range ct.trackers {
		if t, ok := t.(DialerConnectionTagger); ok {
			ctx = t.TagDialerConnection(ctx, info)
		}
	}

	return ctx
}

func (ct chainConnectionTracker) TagListenerConnection(ctx context.Context, info *ListenerConnectionTagInfo) context.Context {
	for _, h := range ct.trackers {
		if t, ok := h.(ListenerConnectionTagger); ok {
			ctx = t.TagListenerConnection(ctx, info)
		}
	}

	return ctx
}

func (ct chainConnectionTracker) TrackConnection(ctx context.Context, stats ConnectionStats) {
	for _, h := range ct.trackers {
		h.TrackConnection(ctx, stats)
	}
}
