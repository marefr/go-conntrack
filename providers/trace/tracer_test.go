// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package trace_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tracetracker "github.com/marefr/go-conntrack/providers/trace"
	"github.com/marefr/go-conntrack/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/trace"
)

const listenerName = "some_name"

func TestDialerTracingCapturedInPage(t *testing.T) {
	listener, tracker := setupServerListener(t)

	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithTrackers(tracker))
	dialerName := "some_dialer"
	conn, err := dialFunc(conntrack.WithDialName(context.Background(), dialerName), "tcp", listener.Addr().String())
	time.Sleep(5 * time.Millisecond)
	require.NoError(t, err, "DialContext should successfully establish a conn here")
	assert.Contains(t, fetchTraceEvents(t, "net.ClientConn."+dialerName), conn.LocalAddr().String(),
		"the /debug/trace/events page must contain the live connection")
	time.Sleep(5 * time.Millisecond)
	conn.Close()

	t.Logf("stopping http.Server at: %v", listener.Addr().String())
	err = listener.Close()
	require.NoError(t, err, "should be able to close serverListener")
}

func TestListenerTracingCapturedInPage(t *testing.T) {
	listener, _ := setupServerListener(t)

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", listener.Addr().String())
	require.NoError(t, err, "DialContext should successfully establish a conn here")
	time.Sleep(5 * time.Millisecond)
	assert.Contains(t, fetchTraceEvents(t, "net.ServerConn."+listenerName), conn.LocalAddr().String(),
		"the /debug/trace/events page must contain the live connection")
	time.Sleep(5 * time.Millisecond)
	conn.Close()

	t.Logf("stopping http.Server at: %v", listener.Addr().String())
	err = listener.Close()
	require.NoError(t, err, "should be able to close serverListener")
}

func setupServerListener(t *testing.T) (net.Listener, conntrack.ConnectionTracker) {
	t.Helper()

	serverListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "must be able to allocate a port for serverListener")

	tracker := tracetracker.New()
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

func fetchTraceEvents(t *testing.T, familyName string) string {
	t.Helper()

	resp := httptest.NewRecorder()
	url := fmt.Sprintf("/debug/events?fam=%s&b=0&exp=1", familyName)
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err, "failed creating request for trace events handler")
	trace.RenderEvents(resp, req, true)
	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed reading the trace page")
	return string(out)
}
