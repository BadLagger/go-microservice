package handlers

import (
	"encoding/json"
	"fmt"
	"go-microservice/models"
	"go-microservice/repository"
	"go-microservice/utils"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	log      *utils.Logger
	repo     *repository.MinIoRepository
	validate validator.Validate
}

func NewUserHandler(repo *repository.MinIoRepository) *UserHandler {
	return &UserHandler{
		log:      utils.GlobalLogger(),
		repo:     repo,
		validate: *validator.New(),
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

func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Get All Users from: %s", r.RemoteAddr)

	users := h.repo.GetAllUsers()

	if users == nil {
		users = []models.User{}
	}

	reponse := map[string]interface{}{
		"user_num": len(users),
		"users":    users,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(reponse); err != nil {
		h.log.Error("Failed to encode response: %+v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *UserHandler) AddNewUser(w http.ResponseWriter, r *http.Request) {
	h.log.Info("Try Add New User From: %s", r.RemoteAddr)

	var userRequest models.UserDto

	err := json.NewDecoder(r.Body).Decode(&userRequest)
	if err != nil {
		h.log.Error("Request parse error: %+v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = h.validateRequest(w, userRequest); err != nil {
		return
	}

	user := h.repo.AddNewUser(userRequest.Name, userRequest.Email, r.Context())
	if user == nil {
		h.log.Error("Cann't create user: %s email: %s", userRequest.Name, userRequest.Email)
		http.Error(w, "Cann't create user", http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		h.log.Error("Failed to encode response: %+v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *UserHandler) validateRequest(w http.ResponseWriter, s interface{}) error {

	log := utils.GlobalLogger()

	if err := h.validate.Struct(s); err != nil {
		var validationErrors []string
		for _, err := range err.(validator.ValidationErrors) {
			log.Error("Field %s failed validator (%s = %s)",
				err.Field(),
				err.Tag(),
				err.Param())
			validationErrors = append(validationErrors, fmt.Sprintf(
				"Field %s failed validator (%s = %s)",
				err.Field(),
				err.Tag(),
				err.Param(),
			))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":   "Validation failed",
			"details": validationErrors,
		})
		return err
	}
	return nil
}
