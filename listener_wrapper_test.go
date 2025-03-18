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
	"github.com/stretchr/testify/suite"
)

func TestListener(t *testing.T) {
	suite.Run(t, &ListenerTestSuite{})
}

const listenerName = "some_name"

type ListenerTestSuite struct {
	suite.Suite

	tracker        *testConnectionTracker
	serverListener net.Listener
	httpServer     http.Server
}

func (s *ListenerTestSuite) SetupTest() {
	var err error
	s.serverListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, "must be able to allocate a port for serverListener")

	s.tracker = newTestConnectionTracker()
	s.serverListener = conntrack.NewListener(s.serverListener,
		conntrack.ListenerWithName(listenerName),
		conntrack.ListenerWithTrackers(s.tracker),
	)
	s.httpServer = http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		_ = s.httpServer.Serve(s.serverListener)
	}()
}

func (s *ListenerTestSuite) TestListenerUnderNormalConnection() {
	beginTime := time.Now()
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")

	time.Sleep(time.Millisecond)

	require.Len(s.T(), s.tracker.stats, 3)
	sAttempt := s.tracker.stats[0].(*conntrack.ConnectionAttempt)
	assert.NotNil(s.T(), sAttempt)
	assert.False(s.T(), sAttempt.Client)
	assert.Equal(s.T(), 1, sAttempt.Attempt)
	assert.NotZero(s.T(), sAttempt.BeginTime)

	sEstablished := s.tracker.stats[1].(*conntrack.ConnectionEstablished)
	assert.NotNil(s.T(), sEstablished)
	assert.False(s.T(), sEstablished.Client)
	assert.Equal(s.T(), 1, sEstablished.Attempts)
	assert.NotZero(s.T(), sEstablished.BeginTime)
	assert.Greater(s.T(), sEstablished.EndTime, sEstablished.BeginTime)
	assert.Equal(s.T(), s.serverListener.Addr().String(), sEstablished.LocalAddr.String())
	assert.NotEmpty(s.T(), sEstablished.LocalAddr.String())

	sAttemptTwo := s.tracker.stats[2].(*conntrack.ConnectionAttempt)
	assert.NotNil(s.T(), sAttemptTwo)
	assert.False(s.T(), sAttemptTwo.Client)
	assert.Equal(s.T(), 1, sAttemptTwo.Attempt)
	assert.Greater(s.T(), sAttemptTwo.BeginTime, beginTime)

	err = conn.Close()
	assert.NoError(s.T(), err)

	time.Sleep(time.Millisecond)

	require.Len(s.T(), s.tracker.stats, 4)
	sClosed := s.tracker.stats[3].(*conntrack.ConnectionClosed)
	assert.NotNil(s.T(), sClosed)
	assert.False(s.T(), sClosed.Client)
	assert.NoError(s.T(), sClosed.Error)
	assert.Greater(s.T(), sClosed.BeginTime, beginTime)
	assert.Greater(s.T(), sClosed.EndTime, sEstablished.BeginTime)
}

func (s *ListenerTestSuite) TestListenerWithContextName() {
	_, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")

	require.NotNil(s.T(), s.tracker.listenerConnectionTagInfo)
	require.Equal(s.T(), listenerName, s.tracker.listenerConnectionTagInfo.ListenerName)

	listenerNameFromCtx := conntrack.ListenerNameFromContext(s.tracker.listenerConnectionTagCtx)
	require.Equal(s.T(), listenerName, listenerNameFromCtx)
}

func (s *ListenerTestSuite) TearDownTest() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}
