package main

import (
	"context"
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
)

func main() {
	log := utils.GlobalLogger().SetLevel(utils.Debug)
	log.Info("App start!!!")
	defer log.Info("App DONE!!!")

	cfg := utils.CfgLoad("Go-Microservice-Docker")
	log.Debug("Config load for app: %s", cfg.AppName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := repository.NewMinIoRepository(cfg.MinioEndpoint, cfg.MinioUser, cfg.MinioPassword, cfg.MinioBucket, cfg.MinioFile, ctx, 5)
	if repo == nil {
		log.Critical("MinIO down!")
		return
	}
	defer repo.Close()

	prometheusMetrics := metrics.NewPrometheusMetrics()
	rateLimiter := utils.NewRateLimiter(1000, 5000)

	userHandlers := handlers.NewUserHandler(repo)

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
	//router.Use(rateLimiter.LimitMiddleware)
	//router.Use(prometheusMetrics.MetricsMiddleware)

	server := &http.Server{
		Addr:    cfg.HostAddress,
		Handler: router,
	}
	serverErr := make(chan error, 1)

	go func() {

		// 1. Эндпоинт для метрик Prometheus (ВАЖНО!!! На проде не должен торчать наружу )
		router.Handle("/metrics", prometheusMetrics.GetHandler()).Methods("GET")

		router.HandleFunc("/test", userHandlers.TestEndpoint).Methods("GET")
		router.HandleFunc("/api/users", userHandlers.GetAllUsers).Methods("GET")
		router.HandleFunc("/api/users", userHandlers.AddNewUser).Methods("POST")

		router.HandleFunc("/api/users/{id}", userHandlers.GetUserById).Methods("GET")
		router.HandleFunc("/api/users/{id}", userHandlers.ChangeUserById).Methods("PUT")
		router.HandleFunc("/api/users/{id}", userHandlers.DeleteById).Methods("DELETE")

		router.NotFoundHandler = http.HandlerFunc(userHandlers.NotFoundEndpoint)

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
		server.Shutdown(ctx)
		log.Info("Signal to escape! Shutdown")
	case err := <-serverErr:
		log.Critical("Server down: %+v", err)
	case <-ctx.Done():
		log.Error("Context DONE! Unexpected!")
	}

}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})
}
