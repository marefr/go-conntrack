package tlsconfig_test

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marefr/go-conntrack/v2"
	"github.com/marefr/go-conntrack/v2/tlsconfig"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestTLSConfigTestSuite(t *testing.T) {
	suite.Run(t, &TLSConfigTestSuite{})
}

var (
	listenerName = "some_name"
)

type TLSConfigTestSuite struct {
	suite.Suite

	serverListener  net.Listener
	httpServer      http.Server
	requests        []*http.Request
	client          *http.Client
	observerFactory *testListenerObserverFactory
}

func (s *TLSConfigTestSuite) SetupSuite() {
	s.requests = []*http.Request{}
	var err error

	tlsServer := httptest.NewUnstartedServer(http.DefaultServeMux)
	tlsServer.EnableHTTP2 = true
	tlsServer.StartTLS()

	s.httpServer = http.Server{
		Handler: http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(http.StatusOK)
		}),
		TLSConfig: tlsServer.TLS,
	}

	tlsConfig, err := tlsconfig.WithHttp2Enabled(tlsServer.TLS)
	require.NoError(s.T(), err, "failed to enable HTTP2")

	s.serverListener = tls.NewListener(tlsServer.Listener, tlsConfig)
	s.observerFactory = newTestListenerObserverFactory()
	s.serverListener = conntrack.NewListener(s.serverListener, conntrack.ListenerWithObserver(s.observerFactory))
	tlsServer.Listener = s.serverListener

	s.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:            tlsServer.TLS.RootCAs,
				InsecureSkipVerify: true,
			},
			ForceAttemptHTTP2: true,
		},
	}

	go func() {
		s.httpServer.Serve(s.serverListener)
	}()
}

func (s *TLSConfigTestSuite) TestListenerObserverIsCreated() {
	resp, err := s.client.Get(fmt.Sprintf("https://%s", s.serverListener.Addr().String()))
	require.NoError(s.T(), err)

	require.Equal(s.T(), "HTTP/2.0", resp.Proto)

	require.Len(s.T(), s.observerFactory.observers, 1)
	o := s.observerFactory.observers[0]

	require.Equal(s.T(), conntrack.DefaultListenerName, o.name)
	require.Equal(s.T(), 1, len(o.acceptAttempts))
	require.Equal(s.T(), 0, len(o.acceptFailures))
	// require.Equal(s.T(), 1, len(o.connectionsAccepted))
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

func (s *TLSConfigTestSuite) TearDownSuite() {
	s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
	s.httpServer.Close()
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
