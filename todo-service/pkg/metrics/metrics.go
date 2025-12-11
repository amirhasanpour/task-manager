package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Metrics struct {
	RequestTotal           *prometheus.CounterVec
	RequestLatency         *prometheus.HistogramVec
	TasksCount             prometheus.Gauge
	TasksCountByStatus     *prometheus.GaugeVec
	TasksCountByPriority   *prometheus.GaugeVec
	CacheHits              prometheus.Counter
	CacheMisses            prometheus.Counter
	DatabaseErrors         prometheus.Counter
	CacheErrors            prometheus.Counter
	ValidationErrors       prometheus.Counter
	logger                 *zap.Logger
}

func NewMetrics(namespace string) *Metrics {
	labels := []string{"service", "method", "endpoint", "status_code"}

	return &Metrics{
		RequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "request_total",
				Help:      "Total number of requests",
			},
			labels,
		),
		RequestLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_latency_histogram",
				Help:      "Request latency in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			labels,
		),
		TasksCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "tasks_count",
				Help:      "Total number of tasks",
			},
		),
		TasksCountByStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "tasks_count_by_status",
				Help:      "Number of tasks by status",
			},
			[]string{"status"},
		),
		TasksCountByPriority: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "tasks_count_by_priority",
				Help:      "Number of tasks by priority",
			},
			[]string{"priority"},
		),
		CacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits",
			},
		),
		CacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses",
			},
		),
		DatabaseErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "database_errors_total",
				Help:      "Total number of database errors",
			},
		),
		CacheErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_errors_total",
				Help:      "Total number of cache errors",
			},
		),
		ValidationErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "validation_errors_total",
				Help:      "Total number of validation errors",
			},
		),
		logger: zap.L().Named("metrics"),
	}
}

func (m *Metrics) RecordRequest(service, method, endpoint string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	m.RequestTotal.WithLabelValues(service, method, endpoint, status).Inc()
	m.RequestLatency.WithLabelValues(service, method, endpoint, status).Observe(duration.Seconds())
}

func (m *Metrics) UpdateTasksCount(count int) {
	m.TasksCount.Set(float64(count))
}

func (m *Metrics) UpdateTasksCountByStatus(status string, count int) {
	m.TasksCountByStatus.WithLabelValues(status).Set(float64(count))
}

func (m *Metrics) UpdateTasksCountByPriority(priority string, count int) {
	m.TasksCountByPriority.WithLabelValues(priority).Set(float64(count))
}

func (m *Metrics) IncrementCacheHits() {
	m.CacheHits.Inc()
}

func (m *Metrics) IncrementCacheMisses() {
	m.CacheMisses.Inc()
}

func (m *Metrics) IncrementDatabaseErrors() {
	m.DatabaseErrors.Inc()
}

func (m *Metrics) IncrementCacheErrors() {
	m.CacheErrors.Inc()
}

func (m *Metrics) IncrementValidationErrors() {
	m.ValidationErrors.Inc()
}

func (m *Metrics) StartMetricsServer(port string) {
	http.Handle("/metrics", promhttp.Handler())
	
	go func() {
		m.logger.Info("Starting metrics server", zap.String("port", port))
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			m.logger.Error("Failed to start metrics server", zap.Error(err))
		}
	}()
}