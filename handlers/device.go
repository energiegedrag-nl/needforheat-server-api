package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type DeviceHandler struct {
	service *services.DeviceService
}

// Create a new DeviceHandler.
func NewDeviceHandler(service *services.DeviceService) *DeviceHandler {
	return &DeviceHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new device.
func (h *DeviceHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request device.Device
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("failed when getting authentication context value").WithLevel(logrus.ErrorLevel)
	}

	if !auth.IsKind(authorization.AccountToken) {
		return NewHandlerError(err, "wrong token kind", http.StatusForbidden).WithMessage("wrong token kind was used")
	}

	device, err := h.service.Create(request.Name, auth.ID, request.ActivationSecret)
	if err != nil {
		if helpers.IsMySQLRecordNotFoundError(err) {
			return NewHandlerError(err, "not found", http.StatusNotFound)
		}

		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}

		if errors.Is(err, services.ErrHashDoesNotMatchType) {
			return NewHandlerError(err, err.Error(), http.StatusBadRequest)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(&device)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for activating a device.
func (h *DeviceHandler) Activate(w http.ResponseWriter, r *http.Request) error {
	var request device.Device
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("device name present")
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("authorization header not present")
	}

	authHeader = strings.Split(authHeader, "Bearer ")[1]

	if authHeader == "" {
		logrus.Info("authorization malformed")
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("authorization malformed")
	}

	d, err := h.service.Activate(request.Name, authHeader)
	if err != nil {
		if errors.Is(err, device.ErrDeviceActivationSecretIncorrect) {
			return NewHandlerError(err, "forbidden", http.StatusForbidden)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	// We don't need to share all uploads.
	d.Uploads = nil

	err = json.NewEncoder(w).Encode(&d)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting device information.
func (h *DeviceHandler) GetDeviceByName(w http.ResponseWriter, r *http.Request) error {
	deviceName := chi.URLParam(r, "device_name")
	if deviceName == "" {
		return NewHandlerError(nil, "device_name not specified", http.StatusBadRequest)
	}

	device, err := h.service.GetByName(deviceName)
	if err != nil {
		return NewHandlerError(err, "device not found", http.StatusNotFound).WithMessage("device not found")
	}

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	accountID, err := h.service.GetAccountByDeviceID(device.ID)
	if err != nil {
		return NewHandlerError(err, "device not found", http.StatusNotFound).WithMessage("device could not be found by ID")
	}

	if auth.ID != accountID {
		return NewHandlerError(err, "device does not belong to account", http.StatusForbidden).WithMessage("request was made for device not owned by account")
	}

	// We don't need to share all uploads.
	device.Uploads = nil

	err = json.NewEncoder(w).Encode(&device)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting device measurements
func (h *DeviceHandler) GetDeviceMeasurements(w http.ResponseWriter, r *http.Request) error {
	deviceName := chi.URLParam(r, "device_name")

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(nil, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	device, err := h.getDeviceByName(deviceName, auth.ID)
	if err != nil {
		return err
	}

	// filters is a map of query parameters with only: property, start & end
	filters := make(map[string]string)
	allowedFilters := []string{"property", "start", "end"}
	for _, v := range allowedFilters {
		val := r.URL.Query().Get(v)

		if val != "" {
			filters[v] = val
		}
	}

	measurements, err := h.service.GetMeasurementsByDeviceID(device.ID, filters)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting measurements")
	}

	err = json.NewEncoder(w).Encode(&measurements)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting device properties
func (h *DeviceHandler) GetDeviceProperties(w http.ResponseWriter, r *http.Request) error {
	deviceName := chi.URLParam(r, "device_name")

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(nil, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	device, err := h.getDeviceByName(deviceName, auth.ID)
	if err != nil {
		return err
	}

	properties, err := h.service.GetPropertiesByDeviceID(device.ID)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting properties")
	}

	err = json.NewEncoder(w).Encode(&properties)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

func (h *DeviceHandler) getDeviceByName(deviceName string, accountId uint) (*device.Device, error) {
	if deviceName == "" {
		return nil, NewHandlerError(nil, "device_name not specified", http.StatusBadRequest)
	}

	device, err := h.service.GetByName(deviceName)
	if err != nil {
		return nil, NewHandlerError(err, "device not found", http.StatusNotFound).WithMessage("device not found")
	}

	deviceAccountId, err := h.service.GetAccountByDeviceID(device.ID)
	if err != nil {
		return nil, NewHandlerError(err, "device not found", http.StatusNotFound).WithMessage("device could not be found by ID")
	}

	if deviceAccountId != accountId {
		return nil, NewHandlerError(nil, "device does not belong to account", http.StatusForbidden).WithMessage("request was made for device not owned by account")
	}

	return &device, nil
}

// Handle API endpoint for getting devices by account including uploads (Former building)
func (h *DeviceHandler) GetDevicesByAccount(w http.ResponseWriter, r *http.Request) error {
	var err error

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		err = errors.New("failed to get authorization context value")
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("failed when getting authentication context value").WithLevel(logrus.ErrorLevel)
	}

	if !auth.IsKind(authorization.AccountToken) {
		err = errors.New("wrong token kind was used")
		return NewHandlerError(err, "wrong token kind", http.StatusForbidden).WithMessage("wrong token kind was used")
	}

	devices, serviceErr := h.service.GetAllByAccount(auth.ID)

	if serviceErr != nil {
		return NewHandlerError(serviceErr, "error in getting devices", http.StatusInternalServerError).WithMessage("error in getting devices").WithLevel(logrus.ErrorLevel)
	}
	err = json.NewEncoder(w).Encode(devices)
	return err
}
