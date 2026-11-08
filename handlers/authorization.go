package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
)

// A Contextkey is the type for a context key.
type contextKey int

// AuthorizationCtxKey is the key for the authorization value that is passed to the context,
// when the authentication middleware is used.
const AuthorizationCtxKey contextKey = 0

type AuthorizationHandler struct {
	service *services.AuthorizationService
}

// Create a new AuthorizationHandler.
func NewAuthorizationHandler(service *services.AuthorizationService) *AuthorizationHandler {
	return &AuthorizationHandler{
		service: service,
	}
}

func (h *AuthorizationHandler) Middleware(kind authorization.AuthKind) func(next Handler) Handler {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization header not present")
			}

			splitHeader := strings.Split(authHeader, "Bearer ")
			if len(splitHeader) != 2 {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization malformed")
			}

			authHeader = splitHeader[1]
			if authHeader == "" {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization malformed")
			}

			auth, err := h.service.ParseTokenToAuthorization(authHeader)
			if err != nil {
				return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage(fmt.Sprintf("error when parsing token: %s", err.Error()))
			}

			if !auth.IsKind(kind) {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("incorrect authorization kind was used to access route")
			}

			// Add the value of audience to the HTTP context with key AuthenticatedID.
			authCtx := context.WithValue(r.Context(), AuthorizationCtxKey, auth)
			r = r.WithContext(authCtx)

			return next(w, r)
		}
	}
}

func (h *AuthorizationHandler) DoubleMiddleware(kind1, kind2 authorization.AuthKind) func(next Handler) Handler {
	return func(next Handler) Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization header not present")
			}

			splitHeader := strings.Split(authHeader, "Bearer ")
			if len(splitHeader) != 2 {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization malformed")
			}

			authHeader = splitHeader[1]
			if authHeader == "" {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("authorization malformed")
			}

			auth, err := h.service.ParseTokenToAuthorization(authHeader)
			if err != nil {
				return NewHandlerError(err, "unauthorized", http.StatusUnauthorized).WithMessage(fmt.Sprintf("error when parsing token: %s", err.Error()))
			}

			if !auth.IsKind(kind1) && !auth.IsKind(kind2) {
				return NewHandlerError(nil, "unauthorized", http.StatusUnauthorized).WithMessage("incorrect authorization kind was used to access route")
			}

			// Add the value of audience to the HTTP context with key AuthenticatedID.
			authCtx := context.WithValue(r.Context(), AuthorizationCtxKey, auth)
			r = r.WithContext(authCtx)

			return next(w, r)
		}
	}
}
