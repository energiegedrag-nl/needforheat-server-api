package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/displaytime"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/sirupsen/logrus"
)

type UploadHandler struct {
	service  *services.UploadService
	location *time.Location
}

// uploadResponse is the stored upload plus the local time offset, which devices use for the time on their screen.
type uploadResponse struct {
	upload.Upload
	displaytime.Offset
}

// Create a new UploadHandler. location is the time zone whose offset is returned with each upload.
func NewUploadHandler(service *services.UploadService, location *time.Location) *UploadHandler {
	return &UploadHandler{
		service:  service,
		location: location,
	}
}

// Handle API endpoint for creating a new upload.
func (h *UploadHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request upload.Upload
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("failed when getting authentication context value")
	}

	if !auth.IsKind(authorization.DeviceToken) && !auth.IsKind(authorization.AccountToken) {
		return NewHandlerError(err, "wrong token kind", http.StatusForbidden).WithMessage("wrong token kind was used")
	}

	// A device can only upload for itself. Its ID comes from the token, not from the request:
	// the device firmware (generic firmware v3.x and v4) does not send instance_id/instance_type,
	// and a client-supplied instance_id must not let one device write into another one.
	if auth.IsKind(authorization.DeviceToken) {
		request.InstanceID = auth.ID
		request.InstanceType = upload.Device
	}

	upload, err := h.service.Create(request.InstanceID, request.InstanceType, request.DeviceTime, request.Measurements)
	if err != nil {
		if errors.Is(err, services.ErrEmptyUpload) {
			return NewHandlerError(err, "empty upload", http.StatusBadRequest)
		}
		if errors.Is(err, services.ErrDeviceReleased) {
			// The firmware erases its pairing on 410 (needforheat-generic-firmware TODO reuse).
			return NewHandlerError(err, "device released", http.StatusGone)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	// We don't need to return all measurements in the upload response.
	upload.Measurements = nil

	err = json.NewEncoder(w).Encode(&uploadResponse{Upload: upload, Offset: displaytime.At(h.location, time.Now())})
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}
