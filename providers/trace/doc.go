// Copyright (c) The go-conntrack Authors.
// Licensed under the Apache License 2.0.

// Package trace provides tracing of Dialer (client) and Listener (server) interactions.
//
// Note: Importing this package automatically enables tracing and exports HTTP interfaces
// on /debug/requests and /debug/events by automatically register with [http.DefaultServeMux].
package trace
