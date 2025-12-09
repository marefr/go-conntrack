// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package conntrack_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/marefr/go-conntrack/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const listenerName = "some_name"

func TestListenerUnderNormalConnection(t *testing.T) {
	listener, tracker := setupServerListener(t)

	beginTime := time.Now()
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err, "DialContext should successfully establish a conn here")

	time.Sleep(time.Millisecond)

	stats := tracker.GetStats()

	require.Len(t, stats, 3)
	sAttempt := stats[0].(*conntrack.ConnectionAttempt)
	assert.NotNil(t, sAttempt)
	assert.False(t, sAttempt.Client)
	assert.Equal(t, 1, sAttempt.Attempt)
	assert.NotZero(t, sAttempt.BeginTime)

	sEstablished := stats[1].(*conntrack.ConnectionEstablished)
	assert.NotNil(t, sEstablished)
	assert.False(t, sEstablished.Client)
	assert.Equal(t, 1, sEstablished.Attempts)
	assert.NotZero(t, sEstablished.BeginTime)
	assert.Greater(t, sEstablished.EndTime, sEstablished.BeginTime)
	assert.Equal(t, listener.Addr().String(), sEstablished.LocalAddr.String())
	assert.NotEmpty(t, sEstablished.LocalAddr.String())

	sAttemptTwo := stats[2].(*conntrack.ConnectionAttempt)
	assert.NotNil(t, sAttemptTwo)
	assert.False(t, sAttemptTwo.Client)
	assert.Equal(t, 1, sAttemptTwo.Attempt)
	assert.Greater(t, sAttemptTwo.BeginTime, beginTime)

	err = conn.Close()
	assert.NoError(t, err)

	time.Sleep(time.Millisecond)

	stats = tracker.GetStats()

	require.Len(t, stats, 4)
	sClosed := stats[3].(*conntrack.ConnectionClosed)
	assert.NotNil(t, sClosed)
	assert.False(t, sClosed.Client)
	assert.NoError(t, sClosed.Error)
	assert.Greater(t, sClosed.BeginTime, beginTime)
	assert.Greater(t, sClosed.EndTime, sEstablished.BeginTime)

	t.Logf("stopping http.Server at: %v", listener.Addr().String())
	err = listener.Close()
	require.NoError(t, err, "should be able to close serverListener")
}

func TestListenerWithContextName(t *testing.T) {
	listener, tracker := setupServerListener(t)

	_, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err, "DialContext should successfully establish a connection here")

	time.Sleep(time.Millisecond)

	tagCtx, tagInfo := tracker.GetListenerTagInfo()
	require.NotNil(t, tagInfo)
	require.Equal(t, listenerName, tagInfo.ListenerName)

	require.NotNil(t, tagCtx)

	listenerNameFromCtx := conntrack.ListenerNameFromContext(tagCtx)
	require.Equal(t, listenerName, listenerNameFromCtx)

	t.Logf("stopping http.Server at: %v", listener.Addr().String())
	err = listener.Close()
	require.NoError(t, err, "should be able to close serverListener")
}

func setupServerListener(t *testing.T) (net.Listener, *testConnectionTracker) {
	t.Helper()

	serverListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "must be able to allocate a port for serverListener")

	tracker := newTestConnectionTracker()
	serverListener = conntrack.NewListener(serverListener,
		conntrack.ListenerWithName(listenerName),
		conntrack.ListenerWithTrackers(tracker),
	)
	httpServer := http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}

	go func() {
		err = httpServer.Serve(serverListener)
		require.NotErrorIs(t, err, http.ErrServerClosed, "http.Server should only be closed normally")
	}()

	return serverListener, tracker
}
