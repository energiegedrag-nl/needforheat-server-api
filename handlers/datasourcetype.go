package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcetype"
	services "github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/sirupsen/logrus"
)

type DataSourceTypeHandler struct {
	service *services.DataSourceTypeService
}

// Create a new DataSourceTypeHandler.
func NewDataSourceTypeHandler(service *services.DataSourceTypeService) *DataSourceTypeHandler {
	return &DataSourceTypeHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new DataSourceType.
func (h *DataSourceTypeHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request datasourcetype.DataSourceType
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	DataSourceType, err := h.service.Create(
		request.TypeInstanceID,
		request.Category,
		request.Precedes,
		request.InstallationManualURL,
		request.FAQURL,
		request.InfoURL,
		request.UploadSchedule,
		request.MeasurementSchedule,
		request.NotificationThreshold,
	)

	if err != nil {
		if helpers.IsMySQLRecordNotFoundError(err) {
			return NewHandlerError(err, "not found", http.StatusNotFound)
		}

		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}

		if strings.Contains(err.Error(), "circular reference detected") {
			return NewHandlerError(err, "circular reference detected", http.StatusBadRequest)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(DataSourceType)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}
