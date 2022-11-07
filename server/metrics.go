//go:build !no_metrics
// +build !no_metrics

package server

import (
	"net/http"

	"github.com/docker/go-metrics"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

// namespacePrefix is the namespace prefix used for prometheus metrics.
const namespacePrefix = "notary_server"

func prometheusOpts(operation string) prometheus.SummaryOpts {
	return prometheus.SummaryOpts{
		Namespace:   namespacePrefix,
		Subsystem:   "http",
		ConstLabels: prometheus.Labels{"operation": operation},
	}
}

// instrumentedHandler instruments a server handler for monitoring with prometheus.
func instrumentedHandler(handlerName string, handler http.Handler) http.Handler {
	return prometheus.InstrumentHandlerFuncWithOpts(prometheusOpts(handlerName), handler.ServeHTTP) //lint:ignore SA1019 TODO update prometheus API
}

// handleMetricsEndpoint registers the /metrics endpoint.
func handleMetricsEndpoint(r *mux.Router) {
	r.Methods("GET").Path("/metrics").Handler(metrics.Handler())
}
