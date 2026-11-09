package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/account"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cockpit"
	"github.com/energiegedrag-nl/needforheat-server-api/repositories"
	"github.com/energiegedrag-nl/needforheat-server-api/services"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MaxInvitationsPerRequest limits how many participant accounts one request may create.
const MaxInvitationsPerRequest = 200

// CockpitHandler serves the /admin routes used by the NeedForHeat Cockpit. All routes need an admin token.
type CockpitHandler struct {
	repository     *repositories.CockpitRepository
	accountService *services.AccountService
	now            func() time.Time
}

func NewCockpitHandler(repository *repositories.CockpitRepository, accountService *services.AccountService) *CockpitHandler {
	return &CockpitHandler{repository: repository, accountService: accountService, now: time.Now}
}

func writeJSON(w http.ResponseWriter, v any) error {
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
	}
	return nil
}

func idParam(r *http.Request, name string) (uint, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, name), 10, 32)
	if err != nil || id == 0 {
		return 0, NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	return uint(id), nil
}

func notFoundOr500(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || helpers.IsMySQLRecordNotFoundError(err) {
		return NewHandlerError(err, "not found", http.StatusNotFound)
	}
	return NewHandlerError(err, "internal server error", http.StatusInternalServerError).WithLevel(logrus.ErrorLevel)
}

// GET /admin/overview: apps, campaigns and data source lists.
func (h *CockpitHandler) Overview(w http.ResponseWriter, r *http.Request) error {
	overview, err := h.repository.Overview(h.now())
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, overview)
}

// GET /admin/campaigns/{campaign_id}/status: per participant and data source the latest upload and its state.
func (h *CockpitHandler) CampaignStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "campaign_id")
	if err != nil {
		return err
	}
	status, err := h.repository.CampaignStatus(id, h.now())
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, status)
}

// PATCH /admin/campaigns/{campaign_id}: change info URL, start time or end time.
func (h *CockpitHandler) UpdateCampaign(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "campaign_id")
	if err != nil {
		return err
	}
	var update cockpit.CampaignUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	if update.StartTime != nil && update.EndTime != nil && *update.EndTime <= *update.StartTime {
		return NewHandlerError(errors.New("end before start"), "end_time must be after start_time", http.StatusBadRequest)
	}
	if err := h.repository.UpdateCampaign(id, update); err != nil {
		if helpers.IsMySQLDuplicateError(err) {
			return NewHandlerError(err, "duplicate", http.StatusBadRequest)
		}
		return notFoundOr500(err)
	}
	status, err := h.repository.CampaignStatus(id, h.now())
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, status.Campaign)
}

// POST /admin/campaigns/{campaign_id}/accounts with {"count": n}: create n participant accounts and return their
// invitation links. The links are shown once; they are not stored.
func (h *CockpitHandler) CreateAccounts(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "campaign_id")
	if err != nil {
		return err
	}
	var request struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	if request.Count < 1 || request.Count > MaxInvitationsPerRequest {
		return NewHandlerError(errors.New("count out of range"), "count must be between 1 and 200", http.StatusBadRequest)
	}
	invitations := []cockpit.Invitation{}
	for i := 0; i < request.Count; i++ {
		a, err := h.accountService.Create(campaign.Campaign{ID: id})
		if err != nil {
			return notFoundOr500(err)
		}
		invitations = append(invitations, cockpit.Invitation{AccountID: a.ID, InvitationURL: a.InvitationURL})
	}
	return writeJSON(w, invitations) // Handler.ServeHTTP answers 200, like the other create routes
}

// POST /admin/accounts/{account_id}/invitation: a new invitation link for a participant who has not activated yet.
func (h *CockpitHandler) NewInvitation(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "account_id")
	if err != nil {
		return err
	}
	a, err := h.accountService.NewInvitation(id)
	if err != nil {
		if errors.Is(err, account.ErrAccountAlreadyActivated) {
			return NewHandlerError(err, "account is already activated", http.StatusConflict)
		}
		return notFoundOr500(err)
	}
	return writeJSON(w, cockpit.Invitation{AccountID: a.ID, InvitationURL: a.InvitationURL})
}
