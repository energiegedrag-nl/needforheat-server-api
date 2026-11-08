package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/apikey"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/go-chi/chi/v5"
)

type APIKeyHandler struct {
	service *services.APIKeyService
}

// Create a new APIKeyHandler.
func NewAPIKeyHandler(service *services.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		service: service,
	}
}

// Handle API endpoint for getting an API Key
func (h *APIKeyHandler) GetAPIKey(w http.ResponseWriter, r *http.Request) error {
	apiName := chi.URLParam(r, "api_name")

	apiKey, err := h.service.Find(apikey.APIKey{APIName: apiName})
	if err != nil {
		return NewHandlerError(err, "api key not found", http.StatusNotFound).WithMessage("api key not found")
	}

	err = json.NewEncoder(w).Encode(apiKey)
	return err
}
