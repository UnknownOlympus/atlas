package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds the metrics for monitoring the geocoding service.
// It includes counters for tasks processed and API errors,
// a histogram for request durations, and a gauge for active workers.
type Metrics struct {
	TaskProcessed  *prometheus.CounterVec   // Counter for the number of tasks processed
	APIErrors      prometheus.Counter       // Counter for the number of API errors
	RequestSeconds *prometheus.HistogramVec // Histogram for tracking request durations
	ActiveWorkers  prometheus.Gauge         // Gauge for the number of active workers
}

// NewMetrics creates a new Metrics instance with the provided Prometheus Registerer.
// It initializes counters, histograms, and gauges for tracking geocoding tasks,
// API errors, request durations, and active workers.
//
// Parameters:
//   - reg: A Prometheus Registerer used to register the metrics.
//
// Returns:
//   - A pointer to the newly created Metrics instance.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		TaskProcessed: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "geocoding_tasks_processed_total",
			Help: "Total number of processed geocoding tasks.",
		}, []string{"status"}),
		APIErrors: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "geocoding_provider_api_errors_total",
			Help: "Total number of errors received from the geocoding provider API.",
		}),
		RequestSeconds: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "geocoding_provider_request_duration_seconds",
			Help:    "Duration of requests to the geocoding provider API.",
			Buckets: prometheus.DefBuckets,
		}, []string{"provider"}),
		ActiveWorkers: promauto.With(reg).NewGauge(prometheus.GaugeOpts{
			Name: "geocoding_active_workers",
			Help: "Current number of active workers processing tasks.",
		}),
	}
}
