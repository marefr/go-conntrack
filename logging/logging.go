// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package logging

import (
	"context"
	"log/slog"

	"github.com/marefr/go-conntrack/v2"
)

// Logger interface for logging messages.
type Logger interface {
	Log(ctx context.Context, level slog.Level, msg string, fields ...slog.Attr)
}

// LoggerFunc is an adapter to allow the use of
// ordinary functions as [Logger]. If f is a function
// with the appropriate signature, LoggerFunc(f) is a
// [Logger] that calls f.
type LoggerFunc func(ctx context.Context, level slog.Level, msg string, fields ...slog.Attr)

// Log calls fn(ctx, level, msg, attrs...).
func (fn LoggerFunc) Log(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	fn(ctx, level, msg, attrs...)
}

// LoggerOption defines a config option you can set on the logging tracker.
type LoggerOption func(*connectionTrackerLogger)

// WithLogger sets the logger used for logging connection tracker interactions.
func WithLogger(logger Logger) LoggerOption {
	return func(ctl *connectionTrackerLogger) {
		ctl.logger = logger
	}
}

// WithAttemptLogLevel sets the logging level used for logging connection attempts.
func WithAttemptLogLevel(lvl slog.Level) LoggerOption {
	return func(ctl *connectionTrackerLogger) {
		ctl.attemptLevel = lvl
	}
}

type connectionTrackerLogger struct {
	logger       Logger
	attemptLevel slog.Level
}

// New returns a new [conntrack.ConnectionTracker] that logs dialer (client) and listener (server) connection interactions.
func New(logger Logger, options ...LoggerOption) conntrack.ConnectionTracker {
	ctl := &connectionTrackerLogger{
		logger:       logger,
		attemptLevel: slog.LevelDebug,
	}

	for _, opt := range options {
		opt(ctl)
	}

	return ctl
}

func (connectionTrackerLogger) TagDialerConnection(ctx context.Context, info *conntrack.DialerConnectionTagInfo) context.Context {
	return withLogAttributes(ctx, &logAttributes{
		attrs: []slog.Attr{
			slog.String("component", "net.ClientConn"),
			slog.String("dialerName", info.DialerName),
			slog.String("addr", info.Addr),
		},
	})
}

func (ct connectionTrackerLogger) TagListenerConnection(ctx context.Context, info *conntrack.ListenerConnectionTagInfo) context.Context {
	return withLogAttributes(ctx, &logAttributes{
		attrs: []slog.Attr{
			slog.String("component", "net.ServerConn"),
			slog.String("listenerName", info.ListenerName),
		},
	})
}

func (ct connectionTrackerLogger) TrackConnection(ctx context.Context, stats conntrack.ConnectionStats) {
	logAttrs := logAttributesFromContext(ctx)
	if logAttrs == nil {
		return
	}

	if stats.IsClient() {
		ct.handleDialerConnection(ctx, stats, logAttrs)
		return
	}

	ct.handleListenerConnection(ctx, stats, logAttrs)
}

func (ct connectionTrackerLogger) handleDialerConnection(ctx context.Context, stats conntrack.ConnectionStats, logAttrs *logAttributes) {
	attrs := logAttrs.attrs

	switch s := stats.(type) {
	case *conntrack.ConnectionAttempt:
		attrs = append(attrs, slog.Int("attempt", s.Attempt))
		ct.logger.Log(ctx, ct.attemptLevel, "Dial attempt", attrs...)

	case *conntrack.ConnectionAttemptFailed:
		attrs = append(attrs,
			slog.Int("attempt", s.Attempt),
			slog.String("reason", string(s.Reason)),
			slog.String("error", s.Error.Error()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)),
		)
		ct.logger.Log(ctx, slog.LevelError, "Dial attempt failed", attrs...)

	case *conntrack.ConnectionFailed:
		attrs = append(attrs,
			slog.Int("attempts", s.Attempts),
			slog.String("reason", string(s.Reason)),
			slog.String("error", s.Error.Error()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)))
		ct.logger.Log(ctx, slog.LevelError, "Dial failed", attrs...)

	case *conntrack.ConnectionEstablished:
		attrs = append(attrs,
			slog.String("remoteAddr", s.RemoteAddr.String()),
			slog.String("localAddr", s.LocalAddr.String()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)),
		)
		ct.logger.Log(ctx, slog.LevelInfo, "Connection established", attrs...)

	case *conntrack.ConnectionClosed:
		attrs = append(attrs,
			slog.Duration("connectionTime", s.EndTime.Sub(s.BeginTime)),
		)
		if s.Error != nil {
			attrs = append(attrs, slog.String("error", s.Error.Error()))
			ct.logger.Log(ctx, slog.LevelError, "Failed to close connection", attrs...)
		} else {
			ct.logger.Log(ctx, slog.LevelInfo, "Connection closed", attrs...)
		}
	}
}

func (ct connectionTrackerLogger) handleListenerConnection(ctx context.Context, stats conntrack.ConnectionStats, logAttrs *logAttributes) {
	attrs := logAttrs.attrs

	switch s := stats.(type) {
	case *conntrack.ConnectionAttempt:
		attrs = append(attrs, slog.Int("attempt", s.Attempt))
		ct.logger.Log(ctx, ct.attemptLevel, "Accept attempt", attrs...)

	case *conntrack.ConnectionAttemptFailed:
		attrs = append(attrs,
			slog.Int("attempt", s.Attempt),
			slog.String("reason", string(s.Reason)),
			slog.String("error", s.Error.Error()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)),
		)
		ct.logger.Log(ctx, slog.LevelError, "Accept attempt failed", attrs...)

	case *conntrack.ConnectionFailed:
		attrs = append(attrs,
			slog.Int("attempts", s.Attempts),
			slog.String("reason", string(s.Reason)),
			slog.String("error", s.Error.Error()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)),
		)
		ct.logger.Log(ctx, slog.LevelError, "Accept failed", attrs...)

	case *conntrack.ConnectionEstablished:
		attrs = append(attrs,
			slog.String("remoteAddr", s.RemoteAddr.String()),
			slog.String("localAddr", s.LocalAddr.String()),
			slog.Duration("took", s.EndTime.Sub(s.BeginTime)),
		)
		ct.logger.Log(ctx, slog.LevelInfo, "Connection accepted", attrs...)

	case *conntrack.ConnectionClosed:
		attrs = append(attrs,
			slog.Duration("connectionTime", s.EndTime.Sub(s.BeginTime)),
		)
		if s.Error != nil {
			attrs = append(attrs, slog.String("error", s.Error.Error()))
			ct.logger.Log(ctx, slog.LevelError, "Failed to close connection", attrs...)
		} else {
			ct.logger.Log(ctx, slog.LevelInfo, "Connection closed", attrs...)
		}
	}
}

type logAttributes struct {
	attrs []slog.Attr
}

type logAttrsCtxKey struct{}

func withLogAttributes(ctx context.Context, logAttrs *logAttributes) context.Context {
	return context.WithValue(ctx, logAttrsCtxKey{}, logAttrs)
}

func logAttributesFromContext(ctx context.Context) *logAttributes {
	if val := ctx.Value(logAttrsCtxKey{}); val != nil {
		return val.(*logAttributes)
	}

	return nil
}
