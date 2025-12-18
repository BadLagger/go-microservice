package main

import (
	"context"
	"go-microservice/handlers"
	"go-microservice/utils"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
)

func main() {
	log := utils.GlobalLogger().SetLevel(utils.Debug)
	log.Info("App start!!!")
	defer log.Info("App DONE!!!")

	cfg := utils.CfgLoad("Go-Microservice")
	log.Debug("Config load for app: %s", cfg.AppName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userHandlers := handlers.NewUserHandler()

	router := mux.NewRouter()
	server := &http.Server{
		Addr:    cfg.HostAddress,
		Handler: router,
	}
	serverErr := make(chan error, 1)

	go func() {
		router.NotFoundHandler = http.HandlerFunc(userHandlers.NotFoundEndpoint)

		router.HandleFunc("/test", userHandlers.TestEndpoint).Methods("GET")

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
