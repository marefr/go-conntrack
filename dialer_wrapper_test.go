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

func TestDialer(t *testing.T) {
	suite.Run(t, &DialerTestSuite{})
}

type DialerTestSuite struct {
	suite.Suite

	serverListener net.Listener
	httpServer     http.Server
}

func (s *DialerTestSuite) SetupSuite() {
	var err error
	s.serverListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, "must be able to allocate a port for serverListener")
	s.httpServer = http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		_ = s.httpServer.Serve(s.serverListener)
	}()
}

func (s *DialerTestSuite) TestDialerUnderNormalConnection() {
	tracker := newTestConnectionTracker()
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("normal_conn"),
		conntrack.DialWithTrackers(tracker),
	)

	beginTime := time.Now()
	conn, err := dialFunc(context.Background(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")

	require.NotNil(s.T(), tracker.dialerConnectionTagInfo)
	require.Equal(s.T(), "normal_conn", tracker.dialerConnectionTagInfo.DialerName)

	require.Len(s.T(), tracker.stats, 2)
	sAttempt := tracker.stats[0].(*conntrack.ConnectionAttempt)
	assert.NotNil(s.T(), sAttempt)
	assert.True(s.T(), sAttempt.Client)
	assert.Equal(s.T(), 1, sAttempt.Attempt)
	assert.Greater(s.T(), sAttempt.BeginTime, beginTime)

	sEstablished := tracker.stats[1].(*conntrack.ConnectionEstablished)
	assert.NotNil(s.T(), sEstablished)
	assert.True(s.T(), sEstablished.Client)
	assert.Equal(s.T(), 1, sEstablished.Attempts)
	assert.Greater(s.T(), sEstablished.BeginTime, beginTime)
	assert.Greater(s.T(), sEstablished.EndTime, sEstablished.BeginTime)
	assert.Equal(s.T(), s.serverListener.Addr().String(), sEstablished.RemoteAddr.String())
	assert.NotEmpty(s.T(), sEstablished.LocalAddr.String())

	err = conn.Close()
	assert.NoError(s.T(), err)

	require.Len(s.T(), tracker.stats, 3)
	sClosed := tracker.stats[2].(*conntrack.ConnectionClosed)
	assert.NotNil(s.T(), sClosed)
	assert.True(s.T(), sClosed.Client)
	assert.NoError(s.T(), sClosed.Error)
	assert.Greater(s.T(), sClosed.BeginTime, beginTime)
	assert.Greater(s.T(), sClosed.EndTime, sEstablished.BeginTime)
}

func (s *DialerTestSuite) TestDialerWithContextName() {
	tracker := newTestConnectionTracker()
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithTrackers(tracker),
	)

	_, err := dialFunc(conntrack.WithDialName(context.Background(), "ctx_conn"), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")

	require.NotNil(s.T(), tracker.dialerConnectionTagInfo)
	require.Equal(s.T(), "ctx_conn", tracker.dialerConnectionTagInfo.DialerName)
	require.Equal(s.T(), s.serverListener.Addr().String(), tracker.dialerConnectionTagInfo.Addr)

	dialerNameFromCtx := conntrack.DialNameFromContext(tracker.dialerConnectionTagCtx)
	require.Equal(s.T(), "ctx_conn", dialerNameFromCtx)
}

func (s *DialerTestSuite) TestDialerResolutionFailure() {
	tracker := newTestConnectionTracker()
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("res_err"),
		conntrack.DialWithTrackers(tracker),
	)

	beginTime := time.Now()
	_, err := dialFunc(context.Background(), "tcp", "dialer.test.wrong.domain.wrong:443")
	require.Error(s.T(), err, "NewDialContextFunc should fail here")

	require.Len(s.T(), tracker.stats, 3)
	sAttempt := tracker.stats[0].(*conntrack.ConnectionAttempt)
	assert.NotNil(s.T(), sAttempt)
	assert.True(s.T(), sAttempt.Client)
	assert.Equal(s.T(), 1, sAttempt.Attempt)
	assert.Greater(s.T(), sAttempt.BeginTime, beginTime)

	sAttemptFailed := tracker.stats[1].(*conntrack.ConnectionAttemptFailed)
	assert.NotNil(s.T(), sAttemptFailed)
	assert.True(s.T(), sAttemptFailed.Client)
	assert.Equal(s.T(), 1, sAttemptFailed.Attempt)
	assert.Error(s.T(), sAttemptFailed.Error)
	assert.Equal(s.T(), conntrack.FailureReasonResolution, sAttemptFailed.Reason)
	assert.Greater(s.T(), sAttemptFailed.BeginTime, beginTime)
	assert.Greater(s.T(), sAttemptFailed.EndTime, sAttemptFailed.BeginTime)

	sConnFailed := tracker.stats[2].(*conntrack.ConnectionFailed)
	assert.NotNil(s.T(), sConnFailed)
	assert.True(s.T(), sConnFailed.Client)
	assert.Error(s.T(), sConnFailed.Error)
	assert.Equal(s.T(), conntrack.FailureReasonResolution, sConnFailed.Reason)
	assert.Equal(s.T(), 1, sConnFailed.Attempts)
	assert.Greater(s.T(), sConnFailed.BeginTime, beginTime)
	assert.Greater(s.T(), sConnFailed.EndTime, sConnFailed.BeginTime)
	assert.Greater(s.T(), sConnFailed.EndTime, sAttemptFailed.BeginTime)
	assert.Greater(s.T(), sConnFailed.EndTime, sAttemptFailed.EndTime)
}

func (s *DialerTestSuite) TestDialerRefusedFailure() {
	tracker := newTestConnectionTracker()
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("ref_err"),
		conntrack.DialWithTrackers(tracker),
	)

	beginTime := time.Now()
	_, err := dialFunc(context.Background(), "tcp", "127.0.0.1:337") // 337 is a cool port, let's hope its unused.
	require.Error(s.T(), err, "NewDialContextFunc should fail here")

	require.Len(s.T(), tracker.stats, 3)
	sConnFailed := tracker.stats[2].(*conntrack.ConnectionFailed)
	assert.NotNil(s.T(), sConnFailed)
	assert.True(s.T(), sConnFailed.Client)
	assert.Error(s.T(), sConnFailed.Error)
	assert.Equal(s.T(), conntrack.FailureReasonConnectionRefused, sConnFailed.Reason)
	assert.Equal(s.T(), 1, sConnFailed.Attempts)
	assert.Greater(s.T(), sConnFailed.BeginTime, beginTime)
	assert.Greater(s.T(), sConnFailed.EndTime, sConnFailed.BeginTime)
}

func (s *DialerTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}
