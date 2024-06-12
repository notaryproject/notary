//go:build no_metrics
// +build no_metrics

package server

import "net/http"

// instrumentedHandler instruments a server handler for monitoring with prometheus.
func instrumentedHandler(_ string, handler http.Handler) http.Handler {
	return handler
}

// handleMetricsEndpoint registers the /metrics endpoint.
func handleMetricsEndpoint(r *interface{}) {}
