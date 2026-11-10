package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/inventory"
	"github.com/energiegedrag-nl/needforheat-server-api/repositories"
	"github.com/go-chi/chi/v5"
)

// InventoryHandler serves the /admin/inventory routes of the flash station. All routes need an admin token.
type InventoryHandler struct {
	repository *repositories.InventoryRepository
	now        func() time.Time
}

func NewInventoryHandler(repository *repositories.InventoryRepository) *InventoryHandler {
	return &InventoryHandler{repository: repository, now: time.Now}
}

func inventoryError(err error) error {
	switch {
	case errors.Is(err, inventory.ErrStillAssigned):
		return NewHandlerError(err, "device is still assigned to a participant; flash again with force to unlink it", http.StatusConflict)
	case errors.Is(err, inventory.ErrInvalidTransition):
		return NewHandlerError(err, "event not allowed in the current state", http.StatusConflict)
	case errors.Is(err, inventory.ErrUnknownEvent), errors.Is(err, repositories.ErrUnknownAccount):
		return NewHandlerError(err, err.Error(), http.StatusBadRequest)
	}
	return notFoundOr500(err)
}

// GET /admin/inventory: all devices, most recently changed first.
func (h *InventoryHandler) List(w http.ResponseWriter, r *http.Request) error {
	devices, err := h.repository.List()
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, devices)
}

// GET /admin/inventory/{device_id}: one device with its history.
func (h *InventoryHandler) Get(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "device_id")
	if err != nil {
		return err
	}
	d, err := h.repository.Get(id)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, d)
}

// GET /admin/inventory/mac/{mac}: the device with this MAC address, or 404 if it was never flashed here.
func (h *InventoryHandler) GetByMAC(w http.ResponseWriter, r *http.Request) error {
	mac, err := inventory.NormalizeMAC(chi.URLParam(r, "mac"))
	if err != nil {
		return NewHandlerError(err, "invalid MAC address", http.StatusBadRequest)
	}
	d, err := h.repository.GetByMAC(mac)
	if err != nil {
		return notFoundOr500(err)
	}
	return writeJSON(w, d)
}

// POST /admin/inventory/flash: record a flashed and verified device (new or known by its MAC address).
func (h *InventoryHandler) RecordFlash(w http.ResponseWriter, r *http.Request) error {
	var f inventory.Flash
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	mac, err := inventory.NormalizeMAC(f.MAC)
	if err != nil {
		return NewHandlerError(err, "invalid MAC address", http.StatusBadRequest)
	}
	f.MAC = mac
	if f.Name != "" && !inventory.NameMatchesMAC(f.Name, mac) {
		return NewHandlerError(errors.New("name does not match MAC"), "name does not match the MAC address", http.StatusBadRequest)
	}
	if f.Firmware == "" || f.FirmwareVersion == "" {
		return NewHandlerError(errors.New("firmware missing"), "firmware and firmware_version are required", http.StatusBadRequest)
	}
	d, err := h.repository.RecordFlash(f, h.now())
	if err != nil {
		return inventoryError(err)
	}
	return writeJSON(w, d)
}

// POST /admin/inventory/{device_id}/events with {"event": "assign"|"ship"|"return"|"erase"|"note", ...}.
func (h *InventoryHandler) Apply(w http.ResponseWriter, r *http.Request) error {
	id, err := idParam(r, "device_id")
	if err != nil {
		return err
	}
	var c inventory.Change
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		return NewHandlerError(err, "bad request", http.StatusBadRequest)
	}
	if c.Event == inventory.EventFlash {
		return NewHandlerError(inventory.ErrUnknownEvent, "use POST /admin/inventory/flash", http.StatusBadRequest)
	}
	if len(c.Details) > 1000 {
		return NewHandlerError(errors.New("details too long"), "details must be at most 1000 characters", http.StatusBadRequest)
	}
	d, err := h.repository.Apply(id, c, h.now())
	if err != nil {
		return inventoryError(err)
	}
	return writeJSON(w, d)
}
