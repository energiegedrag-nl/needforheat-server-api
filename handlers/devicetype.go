package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/devicetype"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/sirupsen/logrus"
)

type DeviceTypeHandler struct {
	service *services.DeviceTypeService
}

// Create a new DeviceTypeHandler.
func NewDeviceTypeHandler(service *services.DeviceTypeService) *DeviceTypeHandler {
	return &DeviceTypeHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new device type.
func (h *DeviceTypeHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request devicetype.DeviceType
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	dt, err := h.service.Create(request.Name)
	if err != nil {
		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(&dt)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}
