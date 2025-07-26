package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	BranchCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "branchlore_branches_total",
		Help: "The total number of branches being tracked",
	})

	DatabaseCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "branchlore_databases_total",
		Help: "The total number of branch databases",
	})

	DatabaseSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "branchlore_database_size_bytes",
		Help: "Size of branch databases in bytes",
	}, []string{"branch"})

	MaintenanceLoopDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "branchlore_maintenance_duration_seconds",
		Help: "Time spent performing maintenance tasks",
	})

	HealthCheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "branchlore_health_check_duration_seconds",
		Help: "Time spent performing health checks",
	})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "branchlore_http_requests_total",
		Help: "The total number of HTTP requests",
	}, []string{"method", "endpoint", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "branchlore_http_request_duration_seconds",
		Help: "The HTTP request latencies in seconds",
	}, []string{"method", "endpoint"})

	GitOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "branchlore_git_operation_duration_seconds",
		Help: "Time spent on git operations",
	}, []string{"operation"})

	DBQueryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "branchlore_db_query_duration_seconds",
		Help: "Time spent executing database queries",
	})

	DBQueryErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "branchlore_db_query_errors_total",
		Help: "The total number of database query errors",
	})
)
