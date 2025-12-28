package metrics

import (
	"go-microservice/analytics"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometeusMetrics struct {
	TotalReqs      *prometheus.CounterVec
	ReqDuration    *prometheus.HistogramVec
	ReqsInProgress *prometheus.GaugeVec
	rollingAvg     prometheus.Gauge
	rollingStd     prometheus.Gauge
	anomalies      prometheus.Counter
	windowSize     prometheus.Gauge
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

	metrics.rollingAvg = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "metrics_rolling_average",
			Help: "Rolling average of last 50 metrics",
		})

	metrics.rollingStd = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "metrics_rolling_stddev",
			Help: "Standard deviation of last 50 metrics",
		})

	metrics.anomalies = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "metrics_anomalies_total",
			Help: "Total number of detected anomalies",
		})

	metrics.windowSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "metrics_window_size",
			Help: "Current window size (0-50)",
		})

	prometheus.MustRegister(metrics.TotalReqs)
	prometheus.MustRegister(metrics.ReqDuration)
	prometheus.MustRegister(metrics.ReqsInProgress)

	prometheus.MustRegister(metrics.rollingAvg)
	prometheus.MustRegister(metrics.rollingStd)
	prometheus.MustRegister(metrics.anomalies)
	prometheus.MustRegister(metrics.windowSize)

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

func (m *PrometeusMetrics) UpdateFromAnalyzer(a *analytics.Analyzer) {
	m.rollingAvg.Set(a.GetCurrentAvg())
	m.rollingStd.Set(a.GetCurrentStd())
	m.windowSize.Set(float64(a.GetWindowSize()))
}

func (m *PrometeusMetrics) IncrementAnomalies() {
	m.anomalies.Inc()
}
