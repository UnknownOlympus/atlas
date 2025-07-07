package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	TaskProcessed  *prometheus.CounterVec
	APIErrors      prometheus.Counter
	RequestSeconds *prometheus.HistogramVec
	ActiveWorkers  prometheus.Gauge
}

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
