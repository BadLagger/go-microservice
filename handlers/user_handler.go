package handlers

import (
	"go-microservice/utils"
	"net/http"
)

type UserHandler struct {
	log *utils.Logger
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		log: utils.GlobalLogger(),
	}
}

func (h *UserHandler) TestEndpoint(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Test Point Request from: %s", r.RemoteAddr)
	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) NotFoundEndpoint(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Warning! Request from: %s to URL: %s", r.RemoteAddr, r.RequestURI)
	w.WriteHeader(http.StatusNotFound)
}
