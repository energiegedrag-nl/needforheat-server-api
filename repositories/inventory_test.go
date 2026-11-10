package repositories

import (
	"errors"
	"testing"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/inventory"
	"gorm.io/gorm"
)

func TestInventoryLifeCycle(t *testing.T) {
	db := testDB(t)
	must(t, db.AutoMigrate(&InventoryDeviceModel{}, &InventoryEventModel{}))
	repo := NewInventoryRepository(db)
	now := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	if _, err := repo.GetByMAC("AA:BB:CC:DD:EE:FF"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("unknown MAC: %v", err)
	}

	flash := inventory.Flash{MAC: "AA:BB:CC:DD:EE:FF", Chip: "ESP32", DeviceType: "P1-Reader", Name: "1A2B-DDEEFF",
		Firmware: "needforheat-p1-reader-firmware", FirmwareVersion: "v3.4.0"}
	d, err := repo.RecordFlash(flash, now)
	must(t, err)
	if d.State != inventory.Flashed || len(d.Events) != 1 || d.InUse != nil {
		t.Fatalf("after flash: %+v", d)
	}

	// Assigning needs an existing account.
	if _, err := repo.Apply(d.ID, inventory.Change{Event: inventory.EventAssign, AccountID: ptr(uint(99))}, now); !errors.Is(err, ErrUnknownAccount) {
		t.Fatalf("assign to unknown account: %v", err)
	}
	app := AppModel{Name: "NeedForHeat", ProvisioningURLTemplate: "https://energiegedrag.nl/invite/<account_activation_token>"}
	must(t, db.Create(&app).Error)
	campaign := CampaignModel{Name: "Proef", AppModelID: app.ID, InfoURL: "https://energiegedrag.nl/proef"}
	must(t, db.Omit("App").Create(&campaign).Error)
	account := AccountModel{CampaignModelID: campaign.ID}
	must(t, db.Omit("Campaign").Create(&account).Error)

	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventAssign, AccountID: &account.ID}, now.Add(time.Minute))
	must(t, err)
	if d.State != inventory.Assigned || d.AccountID == nil || *d.AccountID != account.ID {
		t.Fatalf("after assign: %+v", d)
	}
	if _, err := repo.RecordFlash(flash, now); !errors.Is(err, inventory.ErrStillAssigned) {
		t.Fatalf("flash while assigned: %v", err)
	}

	// The app activated the device: the inventory shows it in use.
	deviceType := DeviceTypeModel{Name: "P1-Reader"}
	must(t, db.Create(&deviceType).Error)
	must(t, db.Omit("DeviceType").Create(&DeviceModel{Name: "1A2B-DDEEFF", DeviceTypeModelID: deviceType.ID, AccountModelID: account.ID, ActivatedAt: ntime(now)}).Error)
	list, err := repo.List()
	must(t, err)
	if len(list) != 1 || list[0].InUse == nil || list[0].InUse.CampaignID != campaign.ID || *list[0].InUse.ActivatedAt != now.Unix() {
		t.Fatalf("list: %+v", list)
	}

	for _, event := range []string{inventory.EventShip, inventory.EventReturn, inventory.EventErase} {
		d, err = repo.Apply(d.ID, inventory.Change{Event: event}, now.Add(2*time.Minute))
		must(t, err)
	}
	if d.State != inventory.Erased || d.AccountID != nil || len(d.Events) != 5 || d.Events[0].Event != inventory.EventErase {
		t.Fatalf("after erase: %+v", d)
	}

	d, err = repo.RecordFlash(flash, now.Add(time.Hour))
	must(t, err)
	if d.State != inventory.Flashed || len(d.Events) != 6 {
		t.Fatalf("after reflash: %+v", d)
	}
}

func ptr[T any](v T) *T { return &v }
