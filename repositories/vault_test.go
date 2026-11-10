package repositories

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestVaultVersions(t *testing.T) {
	db := testDB(t)
	must(t, db.AutoMigrate(&VaultBlobModel{}, &CampaignMailModel{}))
	repo := NewVaultRepository(db)
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	id := "0b6f6a3e-6c0b-4e55-9a51-2f6f1c9e7d10"
	blob := func(n int) []byte { return []byte(fmt.Sprintf("age-encryption.org/v1\n-> X25519 x\nversie %d", n)) }

	if _, err := repo.Latest(id); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("empty vault: %v", err)
	}
	if _, err := repo.Put(id, 0, []byte("SQLite format 3\x00 plain"), now); !errors.Is(err, ErrVaultNotEncrypted) {
		t.Fatalf("plaintext accepted: %v", err)
	}
	if _, err := repo.Put("../etc", 0, blob(1), now); !errors.Is(err, ErrVaultID) {
		t.Fatalf("bad id accepted: %v", err)
	}
	v, err := repo.Put(id, 0, blob(1), now)
	must(t, err)
	if v != 1 {
		t.Fatalf("first version %d", v)
	}
	if _, err := repo.Put(id, 0, blob(2), now); !errors.Is(err, ErrVaultConflict) {
		t.Fatalf("stale base accepted: %v", err)
	}
	for i := 2; i <= VaultKeep+3; i++ {
		v, err = repo.Put(id, i-1, blob(i), now.Add(time.Duration(i)*time.Minute))
		must(t, err)
	}
	latest, err := repo.Latest(id)
	must(t, err)
	if latest.Version != VaultKeep+3 || string(latest.Data) != string(blob(VaultKeep+3)) || len(latest.SHA256) != 64 {
		t.Fatalf("latest: %d %q", latest.Version, latest.Data)
	}
	versions, err := repo.Versions(id)
	must(t, err)
	if len(versions) != VaultKeep || versions[0].Version != VaultKeep+3 || versions[len(versions)-1].Version != 4 {
		t.Fatalf("versions: %+v", versions)
	}
	if _, err := repo.Get(id, 3); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("old version kept: %v", err)
	}
}

func TestCampaignMail(t *testing.T) {
	db := testDB(t)
	must(t, db.AutoMigrate(&VaultBlobModel{}, &CampaignMailModel{}))
	repo := NewVaultRepository(db)
	app := AppModel{Name: "NeedForHeat", ProvisioningURLTemplate: "https://x/<account_activation_token>"}
	must(t, db.Create(&app).Error)
	campaign := CampaignModel{Name: "Winter", AppModelID: app.ID, InfoURL: "https://x/winter"}
	must(t, db.Omit("App").Create(&campaign).Error)

	if _, err := repo.CampaignMail(999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown campaign: %v", err)
	}
	c, err := repo.CampaignMail(campaign.ID)
	must(t, err)
	if c.Enabled || c.UpdatedAt != nil {
		t.Fatalf("default should be disabled: %+v", c)
	}
	c, err = repo.SetCampaignMail(CampaignMail{CampaignID: campaign.ID, Enabled: true, HelpdeskAlias: "helpdesk-winter@energiegedrag.nl"}, time.Now())
	must(t, err)
	if !c.Enabled || c.HelpdeskAlias != "helpdesk-winter@energiegedrag.nl" || c.UpdatedAt == nil {
		t.Fatalf("after set: %+v", c)
	}
	c, err = repo.SetCampaignMail(CampaignMail{CampaignID: campaign.ID}, time.Now())
	must(t, err)
	if c.Enabled || c.HelpdeskAlias != "" {
		t.Fatalf("after disable: %+v", c)
	}
}
