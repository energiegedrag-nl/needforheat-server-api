package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquery"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type EnergyQueryHandler struct {
	service *services.EnergyQueryService
}

// Create a new EnergyQueryHandler.
func NewEnergyQueryHandler(service *services.EnergyQueryService) *EnergyQueryHandler {
	return &EnergyQueryHandler{
		service: service,
	}
}

// Handle API endpoint for creating a new energy query.
func (h *EnergyQueryHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request energyquery.EnergyQuery
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

	energyQuery, err := h.service.Create(request.EnergyQueryType, auth.ID, request.Uploads)
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

	err = json.NewEncoder(w).Encode(&energyQuery)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting EnergyQuery information.
func (h *EnergyQueryHandler) GetEnergyQueryByName(w http.ResponseWriter, r *http.Request) error {
	queryType := chi.URLParam(r, "energy_query_type")
	if queryType == "" {
		return NewHandlerError(nil, "energy_query_type not specified", http.StatusBadRequest)
	}

	var err error
	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		err = errors.New("failed when getting authentication context value")
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	energyQuery, err := h.service.GetByTypeAndAccount(energyquerytype.EnergyQueryType{EnergyQueryVariety: queryType}, auth.ID)
	if err != nil {
		return NewHandlerError(err, "EnergyQuery not found", http.StatusNotFound).WithMessage("EnergyQuery not found")
	}

	// We don't need to share all uploads.
	energyQuery.Uploads = nil

	err = json.NewEncoder(w).Encode(&energyQuery)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting EnergyQuery measurements
func (h *EnergyQueryHandler) GetEnergyQueryMeasurements(w http.ResponseWriter, r *http.Request) error {
	queryType := chi.URLParam(r, "energy_query_type")
	if queryType == "" {
		return NewHandlerError(nil, "energy_query_type not specified", http.StatusBadRequest)
	}

	var err error
	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		err = errors.New("failed when getting authentication context value")
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	EnergyQuery, err := h.getEnergyQueryByName(energyquerytype.EnergyQueryType{EnergyQueryVariety: queryType}, auth.ID)
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

	measurements, err := h.service.GetMeasurementsByEnergyQueryID(EnergyQuery.ID, filters)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting measurements")
	}

	err = json.NewEncoder(w).Encode(&measurements)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting EnergyQuery properties
func (h *EnergyQueryHandler) GetEnergyQueryProperties(w http.ResponseWriter, r *http.Request) error {
	queryType := chi.URLParam(r, "energy_query_type")
	if queryType == "" {
		return NewHandlerError(nil, "energy_query_type not specified", http.StatusBadRequest)
	}

	var err error
	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		err = errors.New("failed when getting authentication context value")
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	EnergyQuery, err := h.getEnergyQueryByName(energyquerytype.EnergyQueryType{EnergyQueryVariety: queryType}, auth.ID)
	if err != nil {
		return err
	}

	properties, err := h.service.GetPropertiesByEnergyQueryID(EnergyQuery.ID)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting properties")
	}

	err = json.NewEncoder(w).Encode(&properties)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

func (h *EnergyQueryHandler) getEnergyQueryByName(energyQueryType energyquerytype.EnergyQueryType, accountId uint) (*energyquery.EnergyQuery, error) {

	EnergyQuery, err := h.service.GetByTypeAndAccount(energyQueryType, accountId)
	if err != nil {
		return nil, NewHandlerError(err, "EnergyQuery not found", http.StatusNotFound).WithMessage("EnergyQuery not found")
	}

	return &EnergyQuery, nil
}

// Handle API endpoint for getting EnergyQueries by account including uploads (Former building)
func (h *EnergyQueryHandler) GetEnergyQueriesByAccount(w http.ResponseWriter, r *http.Request) error {
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

	energyQueries, serviceErr := h.service.GetAllByAccount(auth.ID)
	if serviceErr != nil {
		return NewHandlerError(serviceErr, "error in getting EnergyQueries", http.StatusInternalServerError).WithMessage("error in getting EnergyQueries").WithLevel(logrus.ErrorLevel)
	}
	err = json.NewEncoder(w).Encode(energyQueries)
	return err
}
