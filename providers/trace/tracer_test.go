package trace_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/marefr/go-conntrack/providers/trace"
	"github.com/marefr/go-conntrack/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const listenerName = "some_name"

func TestTracing(t *testing.T) {
	suite.Run(t, &TracingTestSuite{})
}

type TracingTestSuite struct {
	suite.Suite

	serverListener net.Listener
	httpServer     http.Server
	tracker        conntrack.ConnectionTracker
}

func (s *TracingTestSuite) SetupSuite() {
	var err error
	s.serverListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, "must be able to allocate a port for serverListener")
	s.tracker = trace.New()
	s.serverListener = conntrack.NewListener(s.serverListener, conntrack.ListenerWithName(listenerName), conntrack.ListenerWithTrackers(s.tracker))
	s.httpServer = http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, _ *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		_ = s.httpServer.Serve(s.serverListener)
	}()
}

func (s *TracingTestSuite) TestDialerTracingCapturedInPage() {
	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithTrackers(s.tracker))
	dialerName := "some_dialer"
	conn, err := dialFunc(conntrack.WithDialName(context.TODO(), dialerName), "tcp", s.serverListener.Addr().String())
	time.Sleep(5 * time.Millisecond)
	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")
	assert.Contains(s.T(), fetchTraceEvents(s.T(), "net.ClientConn."+dialerName), conn.LocalAddr().String(),
		"the /debug/trace/events page must contain the live connection")
	time.Sleep(5 * time.Millisecond)
	conn.Close()
}

func (s *TracingTestSuite) TestListenerTracingCapturedInPage() {
	conn, err := (&net.Dialer{}).DialContext(context.TODO(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")
	time.Sleep(5 * time.Millisecond)
	assert.Contains(s.T(), fetchTraceEvents(s.T(), "net.ServerConn."+listenerName), conn.LocalAddr().String(),
		"the /debug/trace/events page must contain the live connection")
	time.Sleep(5 * time.Millisecond)
	conn.Close()
}

func (s *TracingTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}
