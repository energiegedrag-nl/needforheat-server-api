package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type CloudFeedHandler struct {
	service *services.CloudFeedService
}

// Create a new CloudFeedHandler.
func NewCloudFeedHandler(service *services.CloudFeedService) *CloudFeedHandler {
	return &CloudFeedHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new cloud feed.
func (h *CloudFeedHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request cloudfeed.CloudFeed
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	_, err = h.service.Create(r.Context(), auth.ID, request.CloudFeedTypeID, request.AuthGrantToken)
	if err != nil {
		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}

		if _, ok := err.(*oauth2.RetrieveError); ok {
			return NewHandlerError(err, "invalid auth code exchange", http.StatusBadRequest)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	return nil
}

type DownloadArgs struct {
	AccountID   uint
	CloudFeedID uint
	StartPeriod needforheat.Time
	EndPeriod   needforheat.Time
}

// Handle RPC endpoint for downloading data from a cloud feed.
func (h *CloudFeedHandler) Download(args DownloadArgs, reply *string) error {
	cfa, err := h.service.Find(cloudfeed.CloudFeed{AccountID: args.AccountID, CloudFeedTypeID: args.CloudFeedID})
	if err != nil {
		return err
	}

	err = h.service.Download(context.Background(), cfa, args.StartPeriod, args.EndPeriod)
	if err != nil {
		return err
	}

	*reply = "Downloaded data from cloud feed. Check server logs for more information."

	return nil
}
