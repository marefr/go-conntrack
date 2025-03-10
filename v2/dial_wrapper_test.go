package conntrack_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/marefr/go-conntrack/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestDialerWrapper(t *testing.T) {
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
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		s.httpServer.Serve(s.serverListener)
	}()
}

func (s *DialerTestSuite) TestDialerWithDefaultObserver() {
	conntrack.NewDialContextFunc()
}

func (s *DialerTestSuite) TestDialerObserverIsCreated() {
	f := newTestDialObserverFactory()
	conntrack.NewDialContextFunc(conntrack.DialWithObserver(f)) // dialer name = default
	conntrack.NewDialContextFunc(conntrack.DialWithName("foobar"), conntrack.DialWithObserver(f))

	require.Len(s.T(), f.observers, 2)
	require.Equal(s.T(), conntrack.DefaultDialerName, f.observers[0].name)
	require.Equal(s.T(), "foobar", f.observers[1].name)
}

func (s *DialerTestSuite) TestMultipleDialerObserverIsCreated() {
	f := newTestDialObserverFactory()
	conntrack.NewDialContextFunc(conntrack.DialWithObserver(conntrack.DialObserverFactories(f, f))) // dialer name = default
	conntrack.NewDialContextFunc(conntrack.DialWithName("foobar"), conntrack.DialWithObserver(conntrack.DialObserverFactories(f, f)))

	require.Len(s.T(), f.observers, 4)
	require.Equal(s.T(), conntrack.DefaultDialerName, f.observers[0].name)
	require.Equal(s.T(), conntrack.DefaultDialerName, f.observers[1].name)
	require.Equal(s.T(), "foobar", f.observers[2].name)
	require.Equal(s.T(), "foobar", f.observers[3].name)
}

func (s *DialerTestSuite) TestDialerUnderNormalConnection() {
	f := newTestDialObserverFactory()
	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithName("normal_conn"), conntrack.DialWithObserver(f))

	require.Len(s.T(), f.observers, 1)
	o := f.observers[0]

	beforeAttempts := len(o.dialAttempts)
	beforeEstablished := len(o.connectionsEstablished)
	beforeClosed := len(o.connectionsClosed)

	conn, err := dialFunc(context.Background(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")
	assert.Equal(s.T(), beforeAttempts+1, len(o.dialAttempts),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished+1, len(o.connectionsEstablished),
		"the established conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeClosed, len(o.connectionsClosed),
		"the closed conn counter must not be incremented after connection was opened")
	conn.Close()
	assert.Equal(s.T(), beforeClosed+1, len(o.connectionsClosed),
		"the closed conn counter must be incremented after connection was closed")
}

func (s *DialerTestSuite) TestDialerWithContextName() {
	f := newTestDialObserverFactory()
	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithObserver(f))

	require.Len(s.T(), f.observers, 1)

	conn, err := dialFunc(conntrack.DialNameToContext(context.Background(), "ctx_conn"), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")

	require.Len(s.T(), f.observers, 2)
	o := f.observers[1]
	require.Equal(s.T(), "ctx_conn", o.name)

	assert.Equal(s.T(), 1, len(o.dialAttempts),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), 1, len(o.connectionsEstablished),
		"the established conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), 0, len(o.connectionsClosed),
		"the closed conn counter must not be incremented after connection was opened")
	conn.Close()
	assert.Equal(s.T(), 1, len(o.connectionsClosed),
		"the closed conn counter must be incremented after connection was closed")
}

func (s *DialerTestSuite) TestDialerResolutionFailure() {
	f := newTestDialObserverFactory()
	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithName("res_err"), conntrack.DialWithObserver(f))

	require.Len(s.T(), f.observers, 1)
	o := f.observers[0]

	beforeAttempts := len(o.dialAttempts)
	beforeDialFailures := len(o.dialFailures)
	beforeEstablished := len(o.connectionsEstablished)
	beforeClosed := len(o.connectionsClosed)

	_, err := dialFunc(context.Background(), "tcp", "dialer.test.wrong.domain.wrong:443")
	require.Error(s.T(), err, "NewDialContextFunc should fail here")
	assert.Equal(s.T(), beforeAttempts+1, len(o.dialAttempts),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished, len(o.connectionsEstablished),
		"the established conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeClosed, len(o.connectionsClosed),
		"the closed conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeDialFailures+1, len(o.dialFailures),
		"the failure counter for resolution error should be incremented")
}

func (s *DialerTestSuite) TestDialerRefusedFailure() {
	f := newTestDialObserverFactory()
	dialFunc := conntrack.NewDialContextFunc(conntrack.DialWithName("ref_err"), conntrack.DialWithObserver(f))

	require.Len(s.T(), f.observers, 1)
	o := f.observers[0]

	beforeAttempts := len(o.dialAttempts)
	beforeDialFailures := len(o.dialFailures)
	beforeEstablished := len(o.connectionsEstablished)
	beforeClosed := len(o.connectionsClosed)

	_, err := dialFunc(context.TODO(), "tcp", "127.0.0.1:337") // 337 is a cool port, let's hope its unused.
	require.Error(s.T(), err, "NewDialContextFunc should fail here")
	assert.Equal(s.T(), beforeAttempts+1, len(o.dialAttempts),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished, len(o.connectionsEstablished),
		"the established conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeClosed, len(o.connectionsClosed),
		"the closed conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeDialFailures+1, len(o.dialFailures),
		"the failure counter for connection refused error should be incremented")
}

func (s *DialerTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}

type testDialObserver struct {
	name                   string
	dialAttempts           []string
	dialFailures           []error
	connectionsEstablished []net.Conn
	connectionsClosed      []error
}

type testDialObserverFactory struct {
	observers []*testDialObserver
}

func newTestDialObserverFactory() *testDialObserverFactory {
	return &testDialObserverFactory{
		observers: []*testDialObserver{},
	}
}

func (f *testDialObserverFactory) NewDialObserver(name string) conntrack.DialObserver {
	o := &testDialObserver{
		name:                   name,
		dialAttempts:           []string{},
		dialFailures:           []error{},
		connectionsEstablished: []net.Conn{},
		connectionsClosed:      []error{},
	}
	f.observers = append(f.observers, o)
	return o
}

func (o *testDialObserver) DialAttempt(ctx context.Context, addr string) {
	o.dialAttempts = append(o.dialAttempts, addr)
}

func (o *testDialObserver) DialFailed(ctx context.Context, err error) {
	o.dialFailures = append(o.dialFailures, err)
}

func (o *testDialObserver) ConnectionEstablished(ctx context.Context, conn net.Conn) {
	o.connectionsEstablished = append(o.connectionsEstablished, conn)
}

func (o *testDialObserver) ConnectionClosed(ctx context.Context, err error) {
	o.connectionsClosed = append(o.connectionsClosed, err)
}
