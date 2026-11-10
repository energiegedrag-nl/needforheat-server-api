package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/repositories"
	"github.com/go-chi/chi/v5"
)

// VaultHandler serves the encrypted vault of the Cockpit (docs/23) and the mail setting per campaign.
type VaultHandler struct {
	repository *repositories.VaultRepository
	now        func() time.Time
}

func NewVaultHandler(repository *repositories.VaultRepository) *VaultHandler {
	return &VaultHandler{repository: repository, now: time.Now}
}

func vaultID(r *http.Request) (string, error) {
	id := chi.URLParam(r, "vault_id")
	if !repositories.ValidVaultID(id) {
		return "", NewHandlerError(repositories.ErrVaultID, "invalid vault id", http.StatusBadRequest)
	}
	return id, nil
}

func writeVault(w http.ResponseWriter, m repositories.VaultBlobModel) error {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Vault-Version", strconv.Itoa(m.Version))
	w.Header().Set("Cache-Control", "no-store")
	_, err := w.Write(m.Data)
	return err
}

// GET /admin/vaults/{vault_id}: the newest version, as the encrypted bytes; X-Vault-Version tells which.
func (h *VaultHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id, err := vaultID(r)
	if err != nil {
		return err
	}
	m, err := h.repository.Latest(id)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeVault(w, m)
}

// GET /admin/vaults/{vault_id}/versions: the stored versions without data.
func (h *VaultHandler) Versions(w http.ResponseWriter, r *http.Request) error {
	id, err := vaultID(r)
	if err != nil {
		return err
	}
	versions, err := h.repository.Versions(id)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, versions)
}

// GET /admin/vaults/{vault_id}/versions/{version}: an older version.
func (h *VaultHandler) GetVersion(w http.ResponseWriter, r *http.Request) error {
	id, err := vaultID(r)
	if err != nil {
		return err
	}
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if err != nil || version < 1 {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	m, err := h.repository.Get(id, version)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeVault(w, m)
}

// PUT /admin/vaults/{vault_id} with the encrypted vault as body and X-Vault-Base-Version: the version it was based on
// (0 for a new vault). 409 when someone stored a newer version in the meantime.
func (h *VaultHandler) Put(w http.ResponseWriter, r *http.Request) error {
	id, err := vaultID(r)
	if err != nil {
		return err
	}
	base, err := strconv.Atoi(r.Header.Get("X-Vault-Base-Version"))
	if err != nil || base < 0 {
		return NewHandlerError(err, "X-Vault-Base-Version header required", http.StatusBadRequest)
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, repositories.VaultMaxSize))
	if err != nil {
		return NewHandlerError(err, "vault too large or unreadable", http.StatusRequestEntityTooLarge)
	}
	version, err := h.repository.Put(id, base, data, h.now())
	switch {
	case errors.Is(err, repositories.ErrVaultConflict):
		return NewHandlerError(err, "vault was changed in the meantime: load the newest version first", http.StatusConflict)
	case errors.Is(err, repositories.ErrVaultNotEncrypted):
		return NewHandlerError(err, "only age-encrypted vaults are accepted", http.StatusBadRequest)
	case err != nil:
		return notFoundOr500(err)
	}
	return writeJSON(w, map[string]int{"version": version})
}

var helpdeskAliasPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]*@[a-z0-9-]+(\.[a-z0-9-]+)+$`)

// GET /admin/campaigns/{campaign_id}/mail: may this campaign use the operator's mail facility, and its helpdesk alias.
func (h *VaultHandler) CampaignMail(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "campaign_id")
	if err != nil {
		return err
	}
	c, err := h.repository.CampaignMail(id)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, c)
}

// PUT /admin/campaigns/{campaign_id}/mail with {"enabled": bool, "helpdesk_alias": "…"}. Admin decision per campaign.
func (h *VaultHandler) SetCampaignMail(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "campaign_id")
	if err != nil {
		return err
	}
	var c repositories.CampaignMail
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	if c.HelpdeskAlias != "" && !helpdeskAliasPattern.MatchString(c.HelpdeskAlias) {
		return NewHandlerError(errors.New("invalid alias"), "helpdesk_alias must be a lowercase e-mail address", http.StatusBadRequest)
	}
	c.CampaignID = id
	out, err := h.repository.SetCampaignMail(c, h.now())
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, out)
}
