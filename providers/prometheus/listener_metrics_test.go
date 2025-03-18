package prometheus_test

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	promtracker "github.com/marefr/go-conntrack/providers/prometheus"
	"github.com/marefr/go-conntrack/v2"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const listenerName = "some_name"

func TestListenerMetrics(t *testing.T) {
	suite.Run(t, &ListenerMetricsTestSuite{})
}

type ListenerMetricsTestSuite struct {
	suite.Suite

	serverListener net.Listener
	httpServer     http.Server
	metrics        *promtracker.ListenerMetrics
	registry       *prometheus.Registry
}

func (s *ListenerMetricsTestSuite) SetupSuite() {
	s.metrics = promtracker.NewListenerMetrics()
	s.registry = prometheus.NewRegistry()
	s.registry.MustRegister(s.metrics)

	var err error
	s.serverListener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(s.T(), err, "must be able to allocate a port for serverListener")
	s.serverListener = conntrack.NewListener(s.serverListener,
		conntrack.ListenerWithName(listenerName),
		conntrack.ListenerWithTrackers(s.metrics.TrackListener()),
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

func (s *ListenerMetricsTestSuite) TestShouldInitializeMetrics() {
	s.metrics.InitializeMetrics("default", listenerName)

	for testID, testCase := range []struct {
		metricName     string
		existingLabels []*io_prometheus_client.LabelPair
	}{
		{"net_conntrack_listener_conn_attempted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", "default"),
		}},
		{"net_conntrack_listener_conn_accepted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", "default"),
		}},
		{"net_conntrack_listener_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", "default"),
			makeLabelPair("reason", conntrack.FailureReasonUnknown.String()),
		}},
		{"net_conntrack_listener_conn_closed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", "default"),
		}},
		{"net_conntrack_listener_conn_open", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", "default"),
		}},
		{"net_conntrack_listener_conn_attempted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", listenerName),
		}},
		{"net_conntrack_listener_conn_accepted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", listenerName),
		}},
		{"net_conntrack_listener_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", listenerName),
		}},
		{"net_conntrack_listener_conn_closed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", listenerName),
		}},
		{"net_conntrack_listener_conn_open", []*io_prometheus_client.LabelPair{
			makeLabelPair("listener_name", listenerName),
		}},
	} {
		m := matchMetricWithLabels(s.T(), s.registry, testCase.metricName, testCase.existingLabels...)
		assert.NotNil(s.T(), m, "metrics must exist for test case %d", testID)
	}
}

func (s *ListenerMetricsTestSuite) TestListenerUnderNormalConnection() {
	labels := []*io_prometheus_client.LabelPair{
		makeLabelPair("listener_name", listenerName),
	}
	beforeAccepted := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_accepted_total", labels...)
	beforeClosed := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_closed_total", labels...)
	beforeOpen := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_open", labels...)

	conn, err := (&net.Dialer{}).DialContext(context.TODO(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "DialContext should successfully establish a conn here")

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeAccepted+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_accepted_total", labels...),
		"the accepted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_closed_total", labels...),
		"the closed conn counter must not be incremented before the connection is closed")
	assert.Equal(s.T(), beforeOpen+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_open", labels...),
		"the open conn must be incremented when the connection is opened")
	err = conn.Close()
	assert.NoError(s.T(), err)

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeClosed+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_closed_total", labels...),
		"the closed conn counter must be incremented after connection was closed")
	assert.Equal(s.T(), beforeOpen, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_listener_conn_open", labels...),
		"the open conn must be decremented when the connection is closed")
}

func (s *ListenerMetricsTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}
