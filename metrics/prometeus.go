package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometeusMetrics struct {
	TotalReqs      *prometheus.CounterVec
	ReqDuration    *prometheus.HistogramVec
	ReqsInProgress *prometheus.GaugeVec
}

// Кастомный ResponseWriter
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func NewPrometheusMetrics() *PrometeusMetrics {
	metrics := PrometeusMetrics{}
	metrics.TotalReqs = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	metrics.ReqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	metrics.ReqsInProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "http_requests_in_progress",
			Help: "Number of requests currently being served",
		},
		[]string{"method", "endpoint"},
	)

	prometheus.MustRegister(metrics.TotalReqs)
	prometheus.MustRegister(metrics.ReqDuration)
	prometheus.MustRegister(metrics.ReqsInProgress)

	return &metrics
}

func (m *PrometeusMetrics) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Отбрасывает запросы от прометея (ВАЖНО!!!! На продакшене не должен торчать наружу)
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		m.ReqsInProgress.WithLabelValues(r.Method, r.URL.Path).Inc()
		defer m.ReqsInProgress.WithLabelValues(r.Method, r.URL.Path).Dec()

		start := time.Now()

		// Создаем кастомный ResponseWriter для отслеживания статуса
		rw := ResponseWriter{
			ResponseWriter: w,
			StatusCode:     http.StatusOK, // дефолтный статус
		}
		next.ServeHTTP(rw, r)
		m.ReqDuration.WithLabelValues(r.Method, r.URL.Path).Observe(time.Since(start).Seconds())

		statusText := http.StatusText(rw.StatusCode)
		if statusText == "" {
			statusText = "unknown"
		}

		m.TotalReqs.WithLabelValues(r.Method, r.URL.Path, statusText).Inc()
	})
}

// GetHandler возвращает handler для /metrics эндпоинта
func (m *PrometeusMetrics) GetHandler() http.Handler {
	return promhttp.Handler()
}
