// Package metrics wires Prometheus instrumentation into the Echo server.
// It uses a dedicated registry (not prometheus.DefaultRegisterer) so the /metrics
// endpoint is fully self-contained and does not inherit any stray metrics from
// third-party libraries that call prometheus.MustRegister on import.
package metrics

import (
	"github.com/labstack/echo-contrib/v5/echoprometheus"
	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Register attaches Prometheus middleware and exposes GET /metrics.
// Must be called before other middleware so every request — including error paths
// and panics caught by Recover — is counted.
//
// Registered collectors and their overhead:
//
//   - ProcessCollector: CPU seconds, RSS, open FDs (syscall-level, near-zero)
//
//   - GoCollector with MetricsScheduler:
//     go_sched_goroutines_goroutines  → current goroutine count; rising trend
//     without matching RPS growth is a goroutine leak signal
//     go_sched_latencies_seconds      → time from goroutine becoming runnable to
//     actually running; p99 > 10ms under normal load indicates goroutine
//     starvation — the closest proxy to channel-block time without enabling
//     the block profiler (SetBlockProfileRate), which adds 5-15 % CPU overhead
//
//   - GoCollector with MetricsGC:
//     go_gc_pauses_seconds            → STW + concurrent pause histogram
//     go_gc_heap_allocs_bytes_total   → allocation pressure counter
//     go_gc_heap_live_bytes           → live object bytes (true heap pressure)
//
//   - MetricsAll also pulls in:
//     go_sync_mutex_wait_total_seconds_total → cumulative time goroutines spent
//     blocked on sync.Mutex / sync.RWMutex; rate() gives contention per second
//     go_cpu_classes_gc_total_cpu_seconds_total → GC CPU cost
//
// All runtime/metrics counters are maintained by the Go runtime regardless;
// exposing them via Prometheus adds no meaningful overhead.
func Register(e *echo.Echo) {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		// OS-level process stats (CPU, RSS, open FDs)
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),

		// Go runtime stats — MetricsAll covers scheduler, GC, memory, CPU classes,
		// and mutex wait via the runtime/metrics package (Go 1.16+).
		// To reduce cardinality in production replace MetricsAll with specific rules:
		//   collectors.MetricsScheduler, collectors.MetricsGC,
		//   collectors.GoRuntimeMetricsRule{Matcher: regexp.MustCompile(`^/sync/.*`)}
		collectors.NewGoCollector(
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsAll),
		),
	)

	mwCfg := echoprometheus.MiddlewareConfig{
		Skipper: func(c *echo.Context) bool {
			return c.Path() == "/metrics"
		},
		DoNotUseRequestPathFor404: true, // prevents cardinality explosion from unknown paths
		Registerer:                reg,
	}
	e.Use(echoprometheus.NewMiddlewareWithConfig(mwCfg))
	e.GET("/metrics", echoprometheus.NewHandlerWithConfig(echoprometheus.HandlerConfig{
		Gatherer: reg,
	}))
}
