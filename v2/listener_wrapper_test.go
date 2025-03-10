package conntrack_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/marefr/go-conntrack/v2"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestListenerTestSuite(t *testing.T) {
	suite.Run(t, &ListenerTestSuite{})
}

var (
	listenerName = "some_name"
)

type ListenerTestSuite struct {
	suite.Suite

	serverListener net.Listener
	httpServer     http.Server
}

func (s *ListenerTestSuite) SetupSuite() {
	var err error
	s.serverListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, "must be able to allocate a port for serverListener")
	s.serverListener = conntrack.NewListener(s.serverListener, conntrack.ListenerWithName(listenerName))
	s.httpServer = http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
	}
	go func() {
		s.httpServer.Serve(s.serverListener)
	}()
}

func (s *ListenerTestSuite) TestListenerWithDefaultObserver() {
	conntrack.NewListener(s.serverListener)
}

func (s *ListenerTestSuite) TestListenerObserverIsCreated() {
	f := newTestListenerObserverFactory()
	conntrack.NewListener(s.serverListener, conntrack.ListenerWithObserver(f)) // dialer name = default
	conntrack.NewListener(s.serverListener, conntrack.ListenerWithName("foobar"), conntrack.ListenerWithObserver(f))

	require.Len(s.T(), f.observers, 2)
	require.Equal(s.T(), conntrack.DefaultListenerName, f.observers[0].name)
	require.Equal(s.T(), "foobar", f.observers[1].name)
}

// func (s *ListenerTestSuite) TestMonitoringNormalConns() {

// 	beforeAccepted := sumCountersForMetricAndLabels(s.T(), "net_conntrack_listener_conn_accepted_total", listenerName)
// 	beforeClosed := sumCountersForMetricAndLabels(s.T(), "net_conntrack_listener_conn_closed_total", listenerName)

// 	conn, err := (&net.Dialer{}).DialContext(context.TODO(), "tcp", s.serverListener.Addr().String())
// 	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")
// 	assert.Equal(s.T(), beforeAccepted+1, sumCountersForMetricAndLabels(s.T(), "net_conntrack_listener_conn_accepted_total", listenerName),
// 		"the accepted conn counter must be incremented after connection was opened")
// 	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), "net_conntrack_listener_conn_closed_total", listenerName),
// 		"the closed conn counter must not be incremented before the connection is closed")
// 	conn.Close()
// 	assert.Equal(s.T(), beforeClosed+1, sumCountersForMetricAndLabels(s.T(), "net_conntrack_listener_conn_closed_total", listenerName),
// 		"the closed conn counter must be incremented after connection was closed")
// }

func (s *ListenerTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}

type testListenerObserver struct {
	name                string
	acceptAttempts      []int
	acceptFailures      []error
	connectionsAccepted []net.Conn
	connectionsClosed   []error
}

type testListenerObserverFactory struct {
	observers []*testListenerObserver
}

func newTestListenerObserverFactory() *testListenerObserverFactory {
	return &testListenerObserverFactory{
		observers: []*testListenerObserver{},
	}
}

func (f *testListenerObserverFactory) NewListenerObserver(name string) conntrack.ListenerObserver {
	o := &testListenerObserver{
		name:                name,
		acceptAttempts:      []int{},
		acceptFailures:      []error{},
		connectionsAccepted: []net.Conn{},
		connectionsClosed:   []error{},
	}
	f.observers = append(f.observers, o)
	return o
}

func (o *testListenerObserver) AcceptAttempt(attempt int) {
	o.acceptAttempts = append(o.acceptAttempts, attempt)
}

func (o *testListenerObserver) AcceptFailed(err error) {
	o.acceptFailures = append(o.acceptFailures, err)
}

func (o *testListenerObserver) ConnectionAccepted(conn net.Conn) {
	o.connectionsAccepted = append(o.connectionsAccepted, conn)
}

func (o *testListenerObserver) ConnectionClosed(err error) {
	o.connectionsClosed = append(o.connectionsClosed, err)
}
