package main

import (
	"context"
	"go-microservice/handlers"
	"go-microservice/repository"
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

	userHandlers := handlers.NewUserHandler(repo)

	router := mux.NewRouter()
	server := &http.Server{
		Addr:    cfg.HostAddress,
		Handler: router,
	}
	serverErr := make(chan error, 1)

	go func() {

		router.HandleFunc("/test", userHandlers.TestEndpoint).Methods("GET")
		router.HandleFunc("/api/users", userHandlers.GetAllUsers).Methods("GET")
		router.HandleFunc("/api/users", userHandlers.AddNewUser).Methods("POST")

		router.HandleFunc("/api/users/{id}", userHandlers.GetUserById).Methods("GET")
		router.HandleFunc("/api/users/{id}", userHandlers.ChangeUserById).Methods("PUT")

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
