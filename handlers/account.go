// Package handlers defines available HTTP handlers
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/account"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type AccountHandler struct {
	accountService *services.AccountService
}

func NewAccountHandler(accountService *services.AccountService) *AccountHandler {
	return &AccountHandler{
		accountService: accountService,
	}
}

// Handle API endpoint for creating a new account.
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) error {
	var request account.Account
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest).WithLevel(logrus.ErrorLevel)
	}

	account, err := h.accountService.Create(request.Campaign)
	if err != nil {
		if helpers.IsMySQLRecordNotFoundError(err) {
			return NewHandlerError(err, "not found", http.StatusNotFound)
		}

		return NewHandlerError(err, "internal server error", http.StatusInternalServerError)
	}

	err = json.NewEncoder(w).Encode(account)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for activating an account.
// This endpoint should be protected with an account activation token.
func (h *AccountHandler) Activate(w http.ResponseWriter, r *http.Request) error {
	var err error

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		err = errors.New("failed to get authorization context value")
		return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage("failed when getting authentication context value")
	}

	if !auth.IsKind(authorization.AccountActivationToken) {
		err = errors.New("wrong token kind was used")
		return NewHandlerError(err, "wrong token kind", http.StatusForbidden).WithMessage("wrong token kind was used")
	}

	a, err := h.accountService.Activate(auth.ID)
	if err != nil {
		if errors.Is(err, account.ErrAccountAlreadyActivated) {
			return NewHandlerError(err, "account already activated", http.StatusBadRequest)
		}

		return NewHandlerError(err, "account activation failed", http.StatusBadRequest)
	}

	err = json.NewEncoder(w).Encode(a)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting account information.
func (h *AccountHandler) GetAccountByID(w http.ResponseWriter, r *http.Request) error {
	accountIDParam := chi.URLParam(r, "account_id")
	if accountIDParam == "" {
		return NewHandlerError(nil, "account_id not specified", http.StatusBadRequest)
	}

	accountID, err := strconv.ParseUint(accountIDParam, 10, 64)
	if err != nil {
		return NewHandlerError(err, "account_id not a number", http.StatusBadRequest)
	}

	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(nil, "internal server error", http.StatusInternalServerError).WithMessage("failed when getting authentication context value")
	}

	if auth.ID != uint(accountID) {
		return NewHandlerError(nil, "id does not correspond to auth", http.StatusForbidden).WithMessage("request was made for another account's info")
	}

	account, err := h.accountService.GetByID(uint(accountID))
	if err != nil {
		return NewHandlerError(err, "not found", http.StatusNotFound)
	}

	err = json.NewEncoder(w).Encode(&account)
	if err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}

	return nil
}

// Handle API endpoint for getting connected cloud feed auths.
func (h *AccountHandler) GetCloudFeedAuthStatuses(w http.ResponseWriter, r *http.Request) error {
	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return InternalServerError(nil).WithMessage("failed when getting authentication context value")
	}

	cloudFeedAuthStatuses, err := h.accountService.GetCloudFeedAuthStatuses(auth.ID)
	if err != nil {
		return InternalServerError(err).WithMessage("failed when getting cloud feed auth statuses")
	}

	err = json.NewEncoder(w).Encode(&cloudFeedAuthStatuses)
	if err != nil {
		return InternalServerError(err).WithLevel(logrus.ErrorLevel)
	}

	return nil
}
