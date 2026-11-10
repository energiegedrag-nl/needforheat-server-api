package repositories

import (
	"errors"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/inventory"
	"gorm.io/gorm"
)

// InventoryDeviceModel is a device in the inventory of the flash station (energiegedrag-cockpit docs/ontwerp.md, § 6).
type InventoryDeviceModel struct {
	ID              uint   `gorm:"primaryKey"`
	MAC             string `gorm:"column:mac;size:17;uniqueIndex;not null"`
	Chip            string `gorm:"size:32"`
	DeviceType      string `gorm:"size:191"`
	Name            string `gorm:"size:32;index"`
	Firmware        string `gorm:"size:191"`
	FirmwareVersion string `gorm:"size:64"`
	State           string `gorm:"size:16;not null"`
	AccountID       *uint
	Note            string `gorm:"size:1024"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (InventoryDeviceModel) TableName() string {
	return "inventory_device"
}

// InventoryEventModel is one step in the history of an inventory device.
type InventoryEventModel struct {
	ID                uint      `gorm:"primaryKey"`
	InventoryDeviceID uint      `gorm:"index;not null"`
	At                time.Time `gorm:"not null"`
	Event             string    `gorm:"size:16;not null"`
	State             string    `gorm:"size:16;not null"`
	AccountID         *uint
	Details           string `gorm:"size:1024"`
}

func (InventoryEventModel) TableName() string {
	return "inventory_event"
}

// ErrUnknownAccount is returned when a device is assigned to an account that does not exist.
var ErrUnknownAccount = errors.New("unknown account")

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (m InventoryDeviceModel) toDevice() inventory.Device {
	return inventory.Device{
		ID:              m.ID,
		MAC:             m.MAC,
		Chip:            m.Chip,
		DeviceType:      m.DeviceType,
		Name:            m.Name,
		Firmware:        m.Firmware,
		FirmwareVersion: m.FirmwareVersion,
		State:           m.State,
		AccountID:       m.AccountID,
		Note:            m.Note,
		CreatedAt:       m.CreatedAt.Unix(),
		UpdatedAt:       m.UpdatedAt.Unix(),
	}
}

// inUse looks up devices with these names in the device table: those were activated by the app.
func (r *InventoryRepository) inUse(names []string) (map[string]*inventory.InUse, error) {
	out := map[string]*inventory.InUse{}
	if len(names) == 0 {
		return out, nil
	}
	var rows []struct {
		Name        string
		AccountID   uint
		CampaignID  uint
		ActivatedAt *needforheat.Time
	}
	err := r.db.Table("device").
		Select("device.name, device.account_id, account.campaign_id, device.activated_at").
		Joins("JOIN account ON account.id = device.account_id").
		Where("device.name IN ? AND device.deleted_at IS NULL", names).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Name] = &inventory.InUse{AccountID: row.AccountID, CampaignID: row.CampaignID, ActivatedAt: cockpitUnix(row.ActivatedAt)}
	}
	return out, nil
}

func cockpitUnix(t *needforheat.Time) *int64 {
	if t == nil {
		return nil
	}
	v := t.Unix()
	return &v
}

func (r *InventoryRepository) withInUse(models []InventoryDeviceModel) ([]inventory.Device, error) {
	names := []string{}
	for _, m := range models {
		if m.Name != "" {
			names = append(names, m.Name)
		}
	}
	used, err := r.inUse(names)
	if err != nil {
		return nil, err
	}
	devices := make([]inventory.Device, 0, len(models))
	for _, m := range models {
		d := m.toDevice()
		if m.Name != "" {
			d.InUse = used[m.Name]
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// List returns all inventory devices, most recently changed first.
func (r *InventoryRepository) List() ([]inventory.Device, error) {
	var models []InventoryDeviceModel
	if err := r.db.Order("updated_at DESC, id DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	return r.withInUse(models)
}

func (r *InventoryRepository) get(tx *gorm.DB, where string, arg any) (inventory.Device, error) {
	var m InventoryDeviceModel
	if err := tx.Where(where, arg).First(&m).Error; err != nil {
		return inventory.Device{}, err
	}
	devices, err := r.withInUse([]InventoryDeviceModel{m})
	if err != nil {
		return inventory.Device{}, err
	}
	d := devices[0]
	var events []InventoryEventModel
	if err := tx.Where("inventory_device_id = ?", m.ID).Order("at DESC, id DESC").Find(&events).Error; err != nil {
		return inventory.Device{}, err
	}
	d.Events = []inventory.Event{}
	for _, e := range events {
		d.Events = append(d.Events, inventory.Event{ID: e.ID, At: e.At.Unix(), Event: e.Event, State: e.State, AccountID: e.AccountID, Details: e.Details})
	}
	return d, nil
}

// Get returns one device with its history.
func (r *InventoryRepository) Get(id uint) (inventory.Device, error) {
	return r.get(r.db, "id = ?", id)
}

// GetByMAC returns one device with its history; mac must be normalized.
func (r *InventoryRepository) GetByMAC(mac string) (inventory.Device, error) {
	return r.get(r.db, "mac = ?", mac)
}

// RecordFlash adds a device to the inventory or updates it after it was flashed. f.MAC must be normalized.
func (r *InventoryRepository) RecordFlash(f inventory.Flash, now time.Time) (inventory.Device, error) {
	var id uint
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var m InventoryDeviceModel
		err := tx.Where("mac = ?", f.MAC).First(&m).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		state, err := inventory.Next(m.State, inventory.EventFlash, f.Force)
		if err != nil {
			return err
		}
		details := f.Firmware + " " + f.FirmwareVersion
		if m.AccountID != nil {
			details += " (was assigned to an account; link removed)"
		}
		m.MAC, m.Chip, m.DeviceType, m.Name = f.MAC, f.Chip, f.DeviceType, f.Name
		m.Firmware, m.FirmwareVersion, m.State, m.AccountID = f.Firmware, f.FirmwareVersion, state, nil
		m.UpdatedAt = now
		if m.ID == 0 {
			m.CreatedAt = now
		}
		if err := tx.Save(&m).Error; err != nil {
			return err
		}
		id = m.ID
		return tx.Create(&InventoryEventModel{InventoryDeviceID: m.ID, At: now, Event: inventory.EventFlash, State: state, Details: details}).Error
	})
	if err != nil {
		return inventory.Device{}, err
	}
	return r.Get(id)
}

// Apply records an event on an existing device.
func (r *InventoryRepository) Apply(id uint, c inventory.Change, now time.Time) (inventory.Device, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var m InventoryDeviceModel
		if err := tx.Where("id = ?", id).First(&m).Error; err != nil {
			return err
		}
		state, err := inventory.Next(m.State, c.Event, false)
		if err != nil {
			return err
		}
		if c.Event == inventory.EventAssign {
			if c.AccountID == nil {
				return ErrUnknownAccount
			}
			var count int64
			if err := tx.Table("account").Where("id = ? AND deleted_at IS NULL", *c.AccountID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return ErrUnknownAccount
			}
			m.AccountID = c.AccountID
		}
		if !inventory.KeepsAccount(state) {
			m.AccountID = nil
		}
		m.State = state
		m.UpdatedAt = now
		if c.Event == inventory.EventNote {
			m.Note = c.Details
		}
		if err := tx.Save(&m).Error; err != nil {
			return err
		}
		return tx.Create(&InventoryEventModel{InventoryDeviceID: m.ID, At: now, Event: c.Event, State: state, AccountID: m.AccountID, Details: c.Details}).Error
	})
	if err != nil {
		return inventory.Device{}, err
	}
	return r.Get(id)
}
