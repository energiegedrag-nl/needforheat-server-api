package repositories

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// The local vault of the Cockpit (energiegedrag-cockpit docs/kluis.md): the campaign manager's participant database, age-encrypted in the browser.
// The server stores it as an opaque blob with a version number and can neither read nor check its contents, apart
// from the age header line that shows it is encrypted.

// VaultKeep is the number of versions kept per vault (the newest ones).
const VaultKeep = 10

// VaultMaxSize limits one vault version.
const VaultMaxSize = 16 << 20

var (
	ErrVaultConflict     = errors.New("vault was changed in the meantime")
	ErrVaultNotEncrypted = errors.New("vault is not an age-encrypted file")
	ErrVaultID           = errors.New("invalid vault id")
	vaultIDPattern       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	ageIntro             = []byte("age-encryption.org/v1\n")
)

// VaultBlobModel is one version of an encrypted vault.
type VaultBlobModel struct {
	ID        uint   `gorm:"primaryKey"`
	VaultID   string `gorm:"size:36;not null;uniqueIndex:idx_vault_version"`
	Version   int    `gorm:"not null;uniqueIndex:idx_vault_version"`
	Data      []byte `gorm:"type:mediumblob;not null"`
	Size      int    `gorm:"not null"`
	SHA256    string `gorm:"column:sha256;size:64;not null"`
	CreatedAt time.Time
}

func (VaultBlobModel) TableName() string {
	return "vault_blob"
}

// CampaignMailModel holds whether a campaign may use the mail facility of the server's operator, set by the admin.
type CampaignMailModel struct {
	CampaignID    uint   `gorm:"primaryKey;autoIncrement:false"`
	Enabled       bool   `gorm:"not null"`
	HelpdeskAlias string `gorm:"size:191"`
	UpdatedAt     time.Time
}

func (CampaignMailModel) TableName() string {
	return "campaign_mail"
}

// VaultVersion describes a stored version without its data.
type VaultVersion struct {
	Version   int    `json:"version"`
	Size      int    `json:"size"`
	SHA256    string `json:"sha256"`
	CreatedAt int64  `json:"created_at"`
}

// CampaignMail is the mail setting of a campaign.
type CampaignMail struct {
	CampaignID    uint   `json:"campaign_id"`
	Enabled       bool   `json:"enabled"`
	HelpdeskAlias string `json:"helpdesk_alias"`
	UpdatedAt     *int64 `json:"updated_at"`
}

type VaultRepository struct {
	db *gorm.DB
}

func NewVaultRepository(db *gorm.DB) *VaultRepository {
	return &VaultRepository{db: db}
}

// ValidVaultID reports whether id is a lowercase UUID (the browser makes it with uuid4).
func ValidVaultID(id string) bool {
	return vaultIDPattern.MatchString(id)
}

// Latest returns the newest version of a vault.
func (r *VaultRepository) Latest(vaultID string) (VaultBlobModel, error) {
	var m VaultBlobModel
	err := r.db.Where("vault_id = ?", vaultID).Order("version DESC").First(&m).Error
	return m, err
}

// Get returns one version of a vault.
func (r *VaultRepository) Get(vaultID string, version int) (VaultBlobModel, error) {
	var m VaultBlobModel
	err := r.db.Where("vault_id = ? AND version = ?", vaultID, version).First(&m).Error
	return m, err
}

// Versions lists the stored versions, newest first.
func (r *VaultRepository) Versions(vaultID string) ([]VaultVersion, error) {
	var models []VaultBlobModel
	if err := r.db.Select("version, size, sha256, created_at").Where("vault_id = ?", vaultID).Order("version DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	out := []VaultVersion{}
	for _, m := range models {
		out = append(out, VaultVersion{Version: m.Version, Size: m.Size, SHA256: m.SHA256, CreatedAt: m.CreatedAt.Unix()})
	}
	return out, nil
}

// Put stores a new version if base is the current version (0 for a new vault) and returns the new version number.
// Older versions beyond VaultKeep are removed.
func (r *VaultRepository) Put(vaultID string, base int, data []byte, now time.Time) (int, error) {
	if !ValidVaultID(vaultID) {
		return 0, ErrVaultID
	}
	if !bytes.HasPrefix(data, ageIntro) {
		return 0, ErrVaultNotEncrypted
	}
	sum := sha256.Sum256(data)
	var version int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var current struct{ Max *int }
		if err := tx.Model(&VaultBlobModel{}).Select("MAX(version) AS max").Where("vault_id = ?", vaultID).Scan(&current).Error; err != nil {
			return err
		}
		latest := 0
		if current.Max != nil {
			latest = *current.Max
		}
		if latest != base {
			return ErrVaultConflict
		}
		version = latest + 1
		if err := tx.Create(&VaultBlobModel{VaultID: vaultID, Version: version, Data: data, Size: len(data), SHA256: hex.EncodeToString(sum[:]), CreatedAt: now}).Error; err != nil {
			return err
		}
		return tx.Where("vault_id = ? AND version <= ?", vaultID, version-VaultKeep).Delete(&VaultBlobModel{}).Error
	})
	if err != nil && !errors.Is(err, ErrVaultConflict) && isDuplicate(err) {
		return 0, ErrVaultConflict // a concurrent save with the same base won
	}
	return version, err
}

// CampaignMail returns the mail setting of a campaign (disabled when never set).
func (r *VaultRepository) CampaignMail(campaignID uint) (CampaignMail, error) {
	var count int64
	if err := r.db.Table("campaign").Where("id = ? AND deleted_at IS NULL", campaignID).Count(&count).Error; err != nil {
		return CampaignMail{}, err
	}
	if count == 0 {
		return CampaignMail{}, gorm.ErrRecordNotFound
	}
	var m CampaignMailModel
	err := r.db.Where("campaign_id = ?", campaignID).First(&m).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return CampaignMail{CampaignID: campaignID}, nil
	}
	if err != nil {
		return CampaignMail{}, err
	}
	at := m.UpdatedAt.Unix()
	return CampaignMail{CampaignID: campaignID, Enabled: m.Enabled, HelpdeskAlias: m.HelpdeskAlias, UpdatedAt: &at}, nil
}

// SetCampaignMail stores the mail setting of a campaign.
func (r *VaultRepository) SetCampaignMail(c CampaignMail, now time.Time) (CampaignMail, error) {
	if _, err := r.CampaignMail(c.CampaignID); err != nil {
		return CampaignMail{}, err
	}
	m := CampaignMailModel{CampaignID: c.CampaignID, Enabled: c.Enabled, HelpdeskAlias: c.HelpdeskAlias, UpdatedAt: now}
	if err := r.db.Save(&m).Error; err != nil {
		return CampaignMail{}, err
	}
	return r.CampaignMail(c.CampaignID)
}

func isDuplicate(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "UNIQUE constraint failed")
}
