// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package conntrack

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jpillora/backoff"
)

const (
	defaultName = "default"
)

type ListenerOptions struct {
	name         string
	tracker      ConnectionTracker
	tcpKeepAlive time.Duration
	retryBackoff *backoff.Backoff
}

type ListenerOption func(*ListenerOptions)

// ListenerWithName sets the name of the Listener for use in tracking and monitoring.
func ListenerWithName(name string) ListenerOption {
	return func(opts *ListenerOptions) {
		opts.name = name
	}
}

// ListenerWithTracker register [ConnectionTracker] trackers that tracks the listener interactions.
func ListenerWithTrackers(trackers ...ConnectionTracker) ListenerOption {
	return func(opts *ListenerOptions) {
		opts.tracker = ChainConnectionTrackers(trackers...)
	}
}

// ListenerWithRetries enables retrying of temporary Accept() errors, with the given backoff between attempts.
// Concurrent accept calls that receive temporary errors have independent backoff scaling.
func ListenerWithRetries(b backoff.Backoff) ListenerOption {
	return func(opts *ListenerOptions) {
		opts.retryBackoff = &b
	}
}

// ListenerWithTCPKeepAlive makes sure that any `net.TCPConn` that get accepted have a keep-alive.
// This is useful for HTTP servers in order for, for example laptops, to not use up resources on the
// server while they don't utilise their connection.
// A value of 0 disables it.
func ListenerWithTCPKeepAlive(keepalive time.Duration) ListenerOption {
	return func(opts *ListenerOptions) {
		opts.tcpKeepAlive = keepalive
	}
}

type connTrackListener struct {
	net.Listener
	opts *ListenerOptions
}

// NewListener returns the given listener wrapped in connection tracking listener.
func NewListener(inner net.Listener, optFuncs ...ListenerOption) net.Listener {
	opts := &ListenerOptions{
		name: defaultName,
	}
	for _, f := range optFuncs {
		f(opts)
	}

	return &connTrackListener{
		Listener: inner,
		opts:     opts,
	}
}

func (ct *connTrackListener) Accept() (net.Conn, error) {
	ctx := WithListenerName(context.Background(), ct.opts.name)
	if t, ok := ct.opts.tracker.(ListenerConnectionTagger); ok {
		ctx = t.TagListenerConnection(ctx, &ListenerConnectionTagInfo{
			ListenerName: ct.opts.name,
		})
	}

	acceptStart := time.Now()
	var (
		conn net.Conn
		err  error
	)

	attempt := 0

	for attempt = 0; ; attempt++ {
		attempBegin := time.Now()
		ct.opts.tracker.TrackConnection(ctx, &ConnectionAttempt{
			Client:    false,
			Attempt:   attempt + 1,
			BeginTime: attempBegin,
		})
		conn, err = ct.Listener.Accept()
		if err == nil || ct.opts.retryBackoff == nil {
			break
		}

		if t, ok := err.(interface{ Temporary() bool }); !ok || !t.Temporary() {
			break
		}
		ct.opts.tracker.TrackConnection(ctx, &ConnectionAttemptFailed{
			Client:    false,
			Attempt:   attempt + 1,
			BeginTime: acceptStart,
			EndTime:   time.Now(),
			Error:     err,
			Reason:    FailureReasonFromError(err),
		})
		time.Sleep(ct.opts.retryBackoff.ForAttempt(float64(attempt)))
	}
	if err != nil {
		ct.opts.tracker.TrackConnection(ctx, &ConnectionFailed{
			Client:    false,
			Attempts:  attempt + 1,
			BeginTime: acceptStart,
			EndTime:   time.Now(),
			Error:     err,
			Reason:    FailureReasonFromError(err),
		})
		return nil, err
	}

	establishedTime := time.Now()
	ct.opts.tracker.TrackConnection(ctx, &ConnectionEstablished{
		Client:     false,
		Attempts:   attempt + 1,
		BeginTime:  acceptStart,
		EndTime:    establishedTime,
		LocalAddr:  conn.LocalAddr(),
		RemoteAddr: conn.RemoteAddr(),
	})

	if tcpConn, ok := conn.(*net.TCPConn); ok && ct.opts.tcpKeepAlive > 0 {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			return nil, fmt.Errorf("failed to enable keep alive: %w", err)
		}

		if err := tcpConn.SetKeepAlivePeriod(ct.opts.tcpKeepAlive); err != nil {
			return nil, fmt.Errorf("failed to set keep alive period: %w", err)
		}
	}

	return newConnectionCloseTracker(ctx, conn, false, ct.opts.tracker, establishedTime), nil
}

type listenerNameKey struct{}

// ListenerNameFromContext returns the name of the listener from the context.
func ListenerNameFromContext(ctx context.Context) string {
	val, ok := ctx.Value(listenerNameKey{}).(string)
	if !ok {
		return ""
	}
	return val
}

// WithListenerName returns a context that will contain a listener name.
func WithListenerName(ctx context.Context, listenerName string) context.Context {
	return context.WithValue(ctx, listenerNameKey{}, listenerName)
}
