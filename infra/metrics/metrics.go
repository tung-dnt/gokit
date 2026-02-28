// Package metrics provides Prometheus instrumentation for the HTTP server.
// It uses a dedicated registry (not prometheus.DefaultRegisterer) so the /metrics
// endpoint is fully self-contained and does not inherit any stray metrics from
// third-party libraries that call prometheus.MustRegister on import.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an http.Handler that serves Prometheus metrics.
//
// Registered collectors and their overhead:
//
//   - ProcessCollector: CPU seconds, RSS, open FDs (syscall-level, near-zero)
//
//   - GoCollector with MetricsAll: scheduler, GC, memory, CPU classes,
//     and mutex wait via the runtime/metrics package (Go 1.16+).
//
// All runtime/metrics counters are maintained by the Go runtime regardless;
// exposing them via Prometheus adds no meaningful overhead.
func Handler() http.Handler {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		),
	)

	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
