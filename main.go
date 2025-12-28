package main

import (
	"context"
	"go-microservice/analytics"
	"go-microservice/handlers"
	"go-microservice/metrics"
	"go-microservice/repository"
	"go-microservice/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
)

func main() {
	log := utils.GlobalLogger().SetLevel(utils.Debug)
	log.Info("App start!!!")
	defer log.Info("App DONE!!!")

	cfg := utils.CfgLoad("Go-Microservice-Docker")
	log.Debug("Config load for app: %s", cfg.AppName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsCtx, metricsCancel := context.WithCancel(context.Background())
	defer metricsCancel()

	redisDb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisEndpoint,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	repo := repository.NewRedisRepository(redisDb)

	analyzer := analytics.NewAnalyzer(50)
	prometheusMetrics := metrics.NewPrometheusMetrics()

	// Запускаем фоновое обновление метрик Prometheus
	go func(ctx context.Context) {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("Metrics updater stopped")
				return
			case <-ticker.C:
				prometheusMetrics.UpdateFromAnalyzer(analyzer)
			}
		}
	}(metricsCtx)

	rateLimiter := utils.NewRateLimiter(1000, 500)

	metricsHandler := handlers.NewMetricsHandler(repo, analyzer, prometheusMetrics)

	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Пропускаем /metrics
			if r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			// Счётчик активных запросов для метрик
			prometheusMetrics.ReqsInProgress.WithLabelValues(r.Method, r.URL.Path).Inc()
			defer prometheusMetrics.ReqsInProgress.WithLabelValues(r.Method, r.URL.Path).Dec()

			if !rateLimiter.Limiter.Allow() {
				// Обработка лимитов по соединению с учётом метрик
				duration := time.Since(start).Seconds()
				prometheusMetrics.ReqDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
				prometheusMetrics.TotalReqs.WithLabelValues(r.Method, r.URL.Path, "Too Many Requests").Inc()
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			rw := &metrics.ResponseWriter{ResponseWriter: w, StatusCode: 200}
			next.ServeHTTP(rw, r)

			duration := time.Since(start).Seconds()
			prometheusMetrics.ReqDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)

			statusText := http.StatusText(rw.StatusCode)
			prometheusMetrics.TotalReqs.WithLabelValues(r.Method, r.URL.Path, statusText).Inc()

		})
	})

	server := &http.Server{
		Addr:    cfg.HostAddress,
		Handler: router,
	}
	serverErr := make(chan error, 1)

	go func() {

		// 1. Эндпоинт для метрик Prometheus (ВАЖНО!!! На проде не должен торчать наружу )
		router.Handle("/metrics", prometheusMetrics.GetHandler()).Methods("GET")

		router.HandleFunc("/metric", metricsHandler.AddMetric).Methods("POST")
		router.HandleFunc("/metrics/latest", metricsHandler.GetLatestMetrics).Methods("GET")
		router.HandleFunc("/health", metricsHandler.HealthCheck).Methods("GET")

		log.Info("Starting server...")
		err := server.ListenAndServe()
		if err != nil {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		metricsCancel()
		server.Shutdown(ctx)
		log.Info("Signal to escape! Shutdown")
	case err := <-serverErr:
		metricsCancel()
		log.Critical("Server down: %+v", err)
	case <-ctx.Done():
		metricsCancel()
		log.Error("Context DONE! Unexpected!")
	}

}
