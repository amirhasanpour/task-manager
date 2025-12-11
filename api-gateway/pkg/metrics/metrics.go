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
	ActiveConnections    prometheus.Gauge
	UserRequests         *prometheus.CounterVec
	TodoRequests         *prometheus.CounterVec
	AuthRequests         *prometheus.CounterVec
	ClientErrors         prometheus.Counter
	ServerErrors         prometheus.Counter
	logger               *zap.Logger
}

func NewMetrics(namespace string) *Metrics {
	labels := []string{"service", "method", "endpoint", "status_code"}

	return &Metrics{
		RequestTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "request_total",
				Help:      "Total number of HTTP requests",
			},
			labels,
		),
		RequestLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_latency_histogram",
				Help:      "HTTP request latency in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			labels,
		),
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active HTTP connections",
			},
		),
		UserRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "user_service_requests_total",
				Help:      "Total number of user service requests",
			},
			[]string{"method", "status"},
		),
		TodoRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "todo_service_requests_total",
				Help:      "Total number of todo service requests",
			},
			[]string{"method", "status"},
		),
		AuthRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_requests_total",
				Help:      "Total number of authentication requests",
			},
			[]string{"method", "status"},
		),
		ClientErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "client_errors_total",
				Help:      "Total number of client errors (4xx)",
			},
		),
		ServerErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "server_errors_total",
				Help:      "Total number of server errors (5xx)",
			},
		),
		logger: zap.L().Named("metrics"),
	}
}

func (m *Metrics) RecordRequest(service, method, endpoint string, statusCode int, duration time.Duration) {
	status := strconv.Itoa(statusCode)
	m.RequestTotal.WithLabelValues(service, method, endpoint, status).Inc()
	m.RequestLatency.WithLabelValues(service, method, endpoint, status).Observe(duration.Seconds())

	// Track error types
	if statusCode >= 400 && statusCode < 500 {
		m.ClientErrors.Inc()
	} else if statusCode >= 500 {
		m.ServerErrors.Inc()
	}
}

func (m *Metrics) RecordUserServiceRequest(method, status string) {
	m.UserRequests.WithLabelValues(method, status).Inc()
}

func (m *Metrics) RecordTodoServiceRequest(method, status string) {
	m.TodoRequests.WithLabelValues(method, status).Inc()
}

func (m *Metrics) RecordAuthRequest(method, status string) {
	m.AuthRequests.WithLabelValues(method, status).Inc()
}

func (m *Metrics) IncrementActiveConnections() {
	m.ActiveConnections.Inc()
}

func (m *Metrics) DecrementActiveConnections() {
	m.ActiveConnections.Dec()
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