package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/authorization"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/propertylabel"
	"github.com/energiegedrag-nl/needforheat-server-api/repositories"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
)

const maxPropertyLabelsBody = 1 << 20

var errNoAuthorization = errors.New("no authorization in the request context")

// PropertyLabelHandler serves the display names of properties: per language, the default
// overridden per campaign (needforheat-gearup-app TODO.md, #graphs and #uploaded).
type PropertyLabelHandler struct {
	repository *repositories.PropertyLabelRepository
	accounts   *services.AccountService
}

func NewPropertyLabelHandler(repository *repositories.PropertyLabelRepository, accounts *services.AccountService) *PropertyLabelHandler {
	return &PropertyLabelHandler{repository: repository, accounts: accounts}
}

// ForAccount handles GET /property_labels?language=nl-NL: the labels for the campaign of the
// account, in the given language (default: the language of the campaign), as
// {"language": "nl-NL", "labels": {"<property name>": "<label>"}}.
func (h *PropertyLabelHandler) ForAccount(w http.ResponseWriter, r *http.Request) error {
	auth, ok := r.Context().Value(AuthorizationCtxKey).(*authorization.Authorization)
	if !ok {
		return NewHandlerError(errNoAuthorization, "unauthorized", http.StatusUnauthorized)
	}
	if !auth.IsKind(authorization.AccountToken) {
		return NewHandlerError(errNoAuthorization, "wrong token kind", http.StatusForbidden).WithMessage("wrong token kind was used")
	}
	account, err := h.accounts.GetByID(auth.ID)
	if err != nil {
		return notFoundOr500(err)
	}
	language := r.URL.Query().Get("language")
	if language == "" {
		language = account.Campaign.Language
	}
	language, err = campaign.NormalizeLanguage(language)
	if err != nil {
		return NewHandlerError(err, err.Error(), http.StatusBadRequest)
	}
	labels, err := h.repository.Resolve(account.Campaign.ID, language)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, map[string]any{"language": language, "labels": labels})
}

// Set handles PUT /property_labels (admin): a list of labels
// ([{"property": "...", "campaign_id": 0, "language": "nl-NL", "label": "..."}]); an empty
// label removes one.
func (h *PropertyLabelHandler) Set(w http.ResponseWriter, r *http.Request) error {
	var labels []propertylabel.PropertyLabel
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxPropertyLabelsBody)).Decode(&labels); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	for i, l := range labels {
		n, err := l.Normalize()
		if err != nil {
			return NewHandlerError(err, err.Error(), http.StatusBadRequest)
		}
		labels[i] = n
	}
	if err := h.repository.Set(labels); err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, map[string]int{"set": len(labels)})
}
