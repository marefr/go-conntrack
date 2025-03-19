// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

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

func TestDialerMetrics(t *testing.T) {
	suite.Run(t, &DialerMetricsTestSuite{})
}

type DialerMetricsTestSuite struct {
	suite.Suite

	serverListener net.Listener
	httpServer     http.Server
	metrics        *promtracker.DialerMetrics
	registry       *prometheus.Registry
}

func (s *DialerMetricsTestSuite) SetupSuite() {
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

	s.metrics = promtracker.NewDialerMetrics()
	s.registry = prometheus.NewRegistry()
	s.registry.MustRegister(s.metrics)
}

func (s *DialerMetricsTestSuite) TestShouldInitializeMetrics() {
	s.metrics.InitializeMetrics("default", "foobar", "something_manual")
	for testID, testCase := range []struct {
		metricName     string
		existingLabels []*io_prometheus_client.LabelPair
	}{
		{"net_conntrack_dialer_conn_attempted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
		}},
		{"net_conntrack_dialer_conn_attempted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "foobar"),
		}},
		{"net_conntrack_dialer_conn_attempted_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "something_manual"),
		}},
		{"net_conntrack_dialer_conn_closed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
		}},
		{"net_conntrack_dialer_conn_closed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "foobar"),
		}},
		{"net_conntrack_dialer_conn_closed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "something_manual"),
		}},
		{"net_conntrack_dialer_conn_established_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
		}},
		{"net_conntrack_dialer_conn_established_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "foobar"),
		}},
		{"net_conntrack_dialer_conn_established_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "something_manual"),
		}},
		{"net_conntrack_dialer_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
			makeLabelPair("reason", conntrack.FailureReasonResolution.String()),
		}},
		{"net_conntrack_dialer_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
			makeLabelPair("reason", conntrack.FailureReasonConnectionRefused.String()),
		}},
		{"net_conntrack_dialer_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
			makeLabelPair("reason", conntrack.FailureReasonTimeout.String()),
		}},
		{"net_conntrack_dialer_conn_failed_total", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
			makeLabelPair("reason", conntrack.FailureReasonUnknown.String()),
		}},
		{"net_conntrack_dialer_conn_open", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "default"),
		}},
		{"net_conntrack_dialer_conn_open", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "foobar"),
		}},
		{"net_conntrack_dialer_conn_open", []*io_prometheus_client.LabelPair{
			makeLabelPair("dialer_name", "something_manual"),
		}},
	} {
		m := matchMetricWithLabels(s.T(), s.registry, testCase.metricName, testCase.existingLabels...)
		assert.NotNil(s.T(), m, "metrics must exist for test case %d", testID)
	}
}

func (s *DialerMetricsTestSuite) TestDialerUnderNormalConnection() {
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("normal_conn"),
		conntrack.DialWithTrackers(s.metrics.TrackDialer()),
	)
	s.metrics.InitializeMetrics("normal_conn")

	labels := []*io_prometheus_client.LabelPair{
		makeLabelPair("dialer_name", "normal_conn"),
	}
	beforeAttempts := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...)
	beforeEstablished := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...)
	beforeClosed := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...)
	beforeOpen := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_open", labels...)

	conn, err := dialFunc(context.TODO(), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeAttempts+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the established conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...),
		"the closed conn counter must not be incremented after connection was opened")
	assert.Equal(s.T(), beforeOpen+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_open", labels...),
		"the open conn gauge must be incremented after connection was opened")
	conn.Close()
	assert.NoError(s.T(), err)

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeClosed+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the closed conn counter must be incremented after connection was closed")
	assert.Equal(s.T(), beforeOpen, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_open", labels...),
		"the open conn gauge must be decremented after connection was closed")
}

func (s *DialerMetricsTestSuite) TestDialerWithContextName() {
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithTrackers(s.metrics.TrackDialer()),
	)
	s.metrics.InitializeMetrics("ctx_conn")

	labels := []*io_prometheus_client.LabelPair{
		makeLabelPair("dialer_name", "ctx_conn"),
	}
	beforeAttempts := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...)
	beforeEstablished := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...)
	beforeClosed := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...)

	conn, err := dialFunc(conntrack.WithDialName(context.TODO(), "ctx_conn"), "tcp", s.serverListener.Addr().String())
	require.NoError(s.T(), err, "NewDialContextFunc should successfully establish a conn here")

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeAttempts+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the established conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...),
		"the closed conn counter must not be incremented after connection was opened")
	err = conn.Close()
	assert.NoError(s.T(), err)

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeClosed+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the closed conn counter must be incremented after connection was closed")
}

func (s *DialerMetricsTestSuite) TestDialerResolutionFailure() {
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("res_err"),
		conntrack.DialWithTrackers(s.metrics.TrackDialer()),
	)
	s.metrics.InitializeMetrics("res_err")

	labels := []*io_prometheus_client.LabelPair{
		makeLabelPair("dialer_name", "res_err"),
	}
	failedLabels := append(labels, makeLabelPair("reason", conntrack.FailureReasonResolution.String()))

	beforeAttempts := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...)
	beforeEstablished := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...)
	beforeResolutionErrors := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_failed_total", failedLabels...)
	beforeClosed := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...)

	_, err := dialFunc(context.TODO(), "tcp", "dialer.test.wrong.domain.wrong:443")
	require.Error(s.T(), err, "NewDialContextFunc should fail here")

	time.Sleep(time.Millisecond)

	assert.Equal(s.T(), beforeAttempts+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the established conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...),
		"the closed conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeResolutionErrors+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_failed_total", failedLabels...),
		"the failure counter for resolution error should be incremented")
}

func (s *DialerMetricsTestSuite) TestDialerRefusedFailure() {
	dialFunc := conntrack.NewDialContextFunc(
		conntrack.DialWithName("ref_err"),
		conntrack.DialWithTrackers(s.metrics.TrackDialer()),
	)

	labels := []*io_prometheus_client.LabelPair{
		makeLabelPair("dialer_name", "ref_err"),
	}
	failedLabels := append(labels, makeLabelPair("reason", conntrack.FailureReasonConnectionRefused.String()))

	beforeAttempts := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...)
	beforeEstablished := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...)
	beforeResolutionErrors := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_failed_total", failedLabels...)
	beforeClosed := sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...)

	_, err := dialFunc(context.TODO(), "tcp", "127.0.0.1:337") // 337 is a cool port, let's hope its unused.
	require.Error(s.T(), err, "NewDialContextFunc should fail here")
	assert.Equal(s.T(), beforeAttempts+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_attempted_total", labels...),
		"the attempted conn counter must be incremented after connection was opened")
	assert.Equal(s.T(), beforeEstablished, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_established_total", labels...),
		"the established conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeClosed, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_closed_total", labels...),
		"the closed conn counter must not be incremented on a failure")
	assert.Equal(s.T(), beforeResolutionErrors+1, sumCountersForMetricAndLabels(s.T(), s.registry, "net_conntrack_dialer_conn_failed_total", failedLabels...),
		"the failure counter for connection refused error should be incremented")
}

func (s *DialerMetricsTestSuite) TearDownSuite() {
	if s.serverListener != nil {
		s.T().Logf("stopped http.Server at: %v", s.serverListener.Addr().String())
		s.serverListener.Close()
	}
}
