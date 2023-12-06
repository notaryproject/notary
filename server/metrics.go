//go:build !no_metrics
// +build !no_metrics

package server

import (
	"net/http"

	"github.com/docker/go-metrics"
	"github.com/gorilla/mux"
)

// namespacePrefix is the namespace prefix used for prometheus metrics.
const namespacePrefix = "notary_server"

// Server uses handlers.Changefeed for two separate routes. It's not allowed
// to register twice ("duplicate metrics collector registration attempted"),
// so checking if it's already instrumented, otherwise skip.
var instrumented = map[string]struct{}{}

// instrumentedHandler instruments a server handler for monitoring with prometheus.
func instrumentedHandler(handlerName string, handler http.Handler) http.Handler {
	if _, registered := instrumented[handlerName]; registered {
		// handler for this operation is already registered.
		return handler
	}
	instrumented[handlerName] = struct{}{}

	// Preserve the old situation, which used ConstLabels: "operation: <operation>"
	// for metrics, but ConstLabels in go-metrics are per-namespace, and use
	// ConstLabels: "handler: <handlerName>" (we pass operationName as handlerName).
	namespace := metrics.NewNamespace(namespacePrefix, "http", metrics.Labels{"operation": handlerName})
	httpMetrics := namespace.NewDefaultHttpMetrics(handlerName)
	metrics.Register(namespace)
	return metrics.InstrumentHandler(httpMetrics, handler)
}

// handleMetricsEndpoint registers the /metrics endpoint.
func handleMetricsEndpoint(r *mux.Router) {
	r.Methods("GET").Path("/metrics").Handler(metrics.Handler())
}
