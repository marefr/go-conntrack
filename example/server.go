// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	promtracker "github.com/marefr/go-conntrack/providers/prometheus"
	"github.com/marefr/go-conntrack/providers/trace"
	"github.com/marefr/go-conntrack/v2"
	"github.com/marefr/go-conntrack/v2/connhelpers"
	"github.com/marefr/go-conntrack/v2/logging"
	"github.com/marefr/go-conntrack/v2/logging/slogadapter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context/ctxhttp"
	_ "golang.org/x/net/trace"
)

var (
	port            = flag.Int("port", 9090, "whether to use tls or not")
	useTLS          = flag.Bool("tls", true, "Whether to use TLS and HTTP2.")
	tlsCertFilePath = flag.String("tls_cert_file", "certs/localhost.crt", "Path to the CRT/PEM file.")
	tlsKeyFilePath  = flag.String("tls_key_file", "certs/localhost.key", "Path to the private key file.")
)

func main() {
	flag.Parse()

	tracingTracker := trace.New()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	loggingTracker := logging.New(slogadapter.Logger(logger))
	dialerMetrics := promtracker.NewDialerMetrics()
	prometheus.MustRegister(dialerMetrics)
	// Since we're using a dynamic name, let's preregister it with prometheus.
	dialerMetrics.InitializeMetrics("google")

	// Make sure all outbound connections use the wrapped dialer.
	http.DefaultTransport.(*http.Transport).DialContext = conntrack.NewDialContextFunc(
		conntrack.DialWithDialer(&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}),
		conntrack.DialWithTrackers(tracingTracker, loggingTracker, dialerMetrics.TrackDialer()),
	)

	handler := func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		rw.Header().Add("Content-Type", "application/json")
		if _, err := rw.Write([]byte(`{"msg": "hello"}`)); err != nil {
			logger.Error("Failed to write response", "error", err)
		}
		callCtx := conntrack.WithDialName(req.Context(), "google")
		resp, err := ctxhttp.Get(callCtx, http.DefaultClient, "https://www.google.comx")
		if err != nil {
			logger.Error("Failed to get response from Google", "error", err)
		} else {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					logger.Error("Failed to close response body", "error", err)
				}
			}()

			logger.Info("Received response from Google", "status", resp.Status)
		}
	}

	http.DefaultServeMux.Handle("/", http.HandlerFunc(handler))
	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())

	httpServer := http.Server{
		Handler: http.DefaultServeMux,
	}
	var httpListener net.Listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	listenerMetrics := promtracker.NewListenerMetrics()
	prometheus.MustRegister(listenerMetrics)
	listenerMetrics.InitializeMetrics("default")
	listener = conntrack.NewListener(listener,
		conntrack.ListenerWithTrackers(tracingTracker, loggingTracker, listenerMetrics.TrackListener()))
	if !*useTLS {
		httpListener = listener
	} else {
		tlsConfig, err := connhelpers.TLSConfigForServerCerts(*tlsCertFilePath, *tlsKeyFilePath)
		if err != nil {
			logger.Error("Failed configuring TLS", "error", err)
			os.Exit(1)
		}
		tlsConfig, err = connhelpers.TLSConfigWithHTTP2Enabled(tlsConfig)
		if err != nil {
			logger.Error("Failed configuring TLS with HTTP2", "error", err)
			os.Exit(1)
		}

		logger.Info("Listening with TLS")
		tlsListener := tls.NewListener(listener, tlsConfig)
		httpListener = tlsListener
	}

	logger.Info("Listening", "addr", listener.Addr().String())
	if err := httpServer.Serve(httpListener); err != nil {
		logger.Error("Failed listening", "error", err)
		os.Exit(1)
	}
}
