package handlers

import (
	"encoding/json"
	"net/http"

	"go-microservice/analytics"
	"go-microservice/metrics"
	"go-microservice/repository"
	"go-microservice/utils"
)

type MetricsHandler struct {
	log      *utils.Logger
	repo     *repository.RedisRepository
	analyzer *analytics.Analyzer
	metrics  *metrics.PrometeusMetrics
}

func NewMetricsHandler(repo *repository.RedisRepository, analyzer *analytics.Analyzer, metrics *metrics.PrometeusMetrics) *MetricsHandler {
	return &MetricsHandler{
		log:      utils.GlobalLogger(),
		repo:     repo,
		analyzer: analyzer,
		metrics:  metrics,
	}
}

// POST /metric - принятие метрики от устройства/IoT
func (h *MetricsHandler) AddMetric(w http.ResponseWriter, r *http.Request) {
	h.log.Info("New metric from: %s", r.RemoteAddr)

	var metric repository.Metric
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		h.log.Error("Parse error: %+v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Базовая валидация (можно расширить)
	if metric.DeviceID == "" {
		h.log.Error("Empty device_id")
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	// Сохраняем в Redis (асинхронно для скорости)
	if err := h.repo.PushMetric(r.Context(), metric); err != nil {
		h.log.Error("Store metric failed: %+v", err)
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	isAnomaly := h.analyzer.AddValue(metric.Value)
	if isAnomaly {
		h.metrics.IncrementAnomalies()
		h.log.Info("Anomaly detected: device=%s value=%.2f avg=%.2f std=%.2f",
			metric.DeviceID, metric.Value,
			h.analyzer.GetCurrentAvg(), h.analyzer.GetCurrentStd())
	}

	w.WriteHeader(http.StatusAccepted) // 202 - принято, обработка асинхронная
}

// GET /metrics/latest - для отладки (посмотреть окно)
func (h *MetricsHandler) GetLatestMetrics(w http.ResponseWriter, r *http.Request) {
	h.log.Debug("Get latest metrics from: %s", r.RemoteAddr)

	metrics, err := h.repo.GetLatestMetrics(r.Context(), 50)
	if err != nil {
		h.log.Error("Fetch metrics failed: %+v", err)
		http.Error(w, "Storage error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		h.log.Error("Encode response failed: %+v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// GET /health - проверка работоспособности (для Kubernetes liveness probe)
func (h *MetricsHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		h.log.Error("Redis ping failed: %+v", err)
		http.Error(w, "Redis unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
