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
	RequestTotal         *prometheus.CounterVec
	RequestLatency       *prometheus.HistogramVec
	UsersCount           prometheus.Gauge
	ActiveUsersCount     prometheus.Gauge
	DatabaseErrors       prometheus.Counter
	AuthenticationErrors prometheus.Counter
	ValidationErrors     prometheus.Counter
	logger              *zap.Logger
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
		UsersCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "users_count",
				Help:      "Total number of users",
			},
		),
		ActiveUsersCount: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_users_count",
				Help:      "Number of active users (logged in last 24 hours)",
			},
		),
		DatabaseErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "database_errors_total",
				Help:      "Total number of database errors",
			},
		),
		AuthenticationErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "authentication_errors_total",
				Help:      "Total number of authentication errors",
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

func (m *Metrics) UpdateUsersCount(count int) {
	m.UsersCount.Set(float64(count))
}

func (m *Metrics) UpdateActiveUsersCount(count int) {
	m.ActiveUsersCount.Set(float64(count))
}

func (m *Metrics) IncrementDatabaseErrors() {
	m.DatabaseErrors.Inc()
}

func (m *Metrics) IncrementAuthenticationErrors() {
	m.AuthenticationErrors.Inc()
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