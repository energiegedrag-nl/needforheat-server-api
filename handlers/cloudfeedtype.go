package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeedtype"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/sirupsen/logrus"
)

type CloudFeedTypeHandler struct {
	service *services.CloudFeedTypeService
}

// Create a new CloudFeedTypeHandler.
func NewCloudFeedTypeHandler(service *services.CloudFeedTypeService) *CloudFeedTypeHandler {
	return &CloudFeedTypeHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new cloud feed.
func (h *CloudFeedTypeHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request cloudfeedtype.CloudFeedType
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	CloudFeedType, err := h.service.Create(request.Name, request.AuthorizationURL, request.TokenURL, request.ClientID, request.ClientSecret, request.Scope, request.RedirectURL)
	if err != nil {
		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(&CloudFeedType)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}
