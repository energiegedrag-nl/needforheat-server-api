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

func TestInventoryReturnReleaseAndReflash(t *testing.T) {
	db := testDB(t)
	must(t, db.AutoMigrate(&InventoryDeviceModel{}, &InventoryEventModel{}))
	repo := NewInventoryRepository(db)
	now := time.Date(2026, 11, 20, 1, 0, 0, 0, time.UTC)

	app := AppModel{Name: "NeedForHeat", ProvisioningURLTemplate: "https://energiegedrag.nl/invite/<account_activation_token>"}
	must(t, db.Create(&app).Error)
	campaign := CampaignModel{Name: "Proef", AppModelID: app.ID, InfoURL: "https://energiegedrag.nl/proef"}
	must(t, db.Omit("App").Create(&campaign).Error)
	first := AccountModel{CampaignModelID: campaign.ID}
	must(t, db.Omit("Campaign").Create(&first).Error)
	second := AccountModel{CampaignModelID: campaign.ID}
	must(t, db.Omit("Campaign").Create(&second).Error)
	deviceType := DeviceTypeModel{Name: "LRM"}
	must(t, db.Create(&deviceType).Error)

	flash := inventory.Flash{MAC: "AA:BB:CC:FE:2E:97", Chip: "ESP32", DeviceType: "LRM", Name: "FE2E-FE2E97",
		Firmware: "needforheat-living-room-module-firmware", FirmwareVersion: "v1.2.0"}
	d, err := repo.RecordFlash(flash, now)
	must(t, err)
	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventAssign, AccountID: &first.ID}, now)
	must(t, err)
	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventShip}, now)
	must(t, err)
	device := DeviceModel{Name: "FE2E-FE2E97", DeviceTypeModelID: deviceType.ID, AccountModelID: first.ID, ActivatedAt: ntime(now)}
	must(t, db.Omit("DeviceType").Create(&device).Error)

	// Return keeps the account; a returned device cannot be flashed without a decision.
	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventReturn}, now.Add(time.Hour))
	must(t, err)
	if d.State != inventory.Returned || d.AccountID == nil || *d.AccountID != first.ID {
		t.Fatalf("after return: %+v", d)
	}
	if _, err := repo.RecordFlash(flash, now.Add(time.Hour)); !errors.Is(err, inventory.ErrStillAssigned) {
		t.Fatalf("flash while returned: %v", err)
	}

	// Repair for the same participant: flash keeps the account, then ship back.
	keep := flash
	keep.KeepAccount = true
	d, err = repo.RecordFlash(keep, now.Add(2*time.Hour))
	must(t, err)
	if d.State != inventory.Returned || d.AccountID == nil || *d.AccountID != first.ID || d.InUse == nil {
		t.Fatalf("after repair flash: %+v", d)
	}
	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventShip}, now.Add(2*time.Hour))
	must(t, err)
	if d.State != inventory.Shipped {
		t.Fatalf("ship back: %+v", d)
	}

	// A shipped device is returned first, then released for another participant.
	if _, err := repo.Apply(d.ID, inventory.Change{Event: inventory.EventRelease}, now); !errors.Is(err, inventory.ErrInvalidTransition) {
		t.Fatalf("release while shipped: %v", err)
	}
	_, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventReturn}, now.Add(3*time.Hour))
	must(t, err)
	release := now.Add(4 * time.Hour)
	d, err = repo.Apply(d.ID, inventory.Change{Event: inventory.EventRelease, Details: "verhuisd"}, release)
	must(t, err)
	if d.State != inventory.Flashed || d.AccountID != nil || d.InUse != nil {
		t.Fatalf("after release: %+v", d)
	}
	var archived DeviceModel
	must(t, db.First(&archived, device.ID).Error)
	if archived.Name != "FE2E-FE2E97~20261120-050000" || archived.ReleasedAt == nil || archived.AccountModelID != first.ID {
		t.Fatalf("archived row: %+v", archived)
	}
	devices := NewDeviceRepository(db)
	if released, err := devices.IsReleased(device.ID); err != nil || !released {
		t.Fatalf("IsReleased: %v, %v", released, err)
	}
	if list, err := devices.GetAllByAccount(first.ID); err != nil || len(list) != 0 {
		t.Fatalf("released device still listed for the account: %+v, %v", list, err)
	}

	// The name is free: the next participant's app activates it.
	next := DeviceModel{Name: "FE2E-FE2E97", DeviceTypeModelID: deviceType.ID, AccountModelID: second.ID, ActivatedAt: ntime(release)}
	must(t, db.Omit("DeviceType").Create(&next).Error)
	if released, err := devices.IsReleased(next.ID); err != nil || released {
		t.Fatalf("new device released: %v, %v", released, err)
	}

	// Releasing by name (also without inventory) archives the second row; nothing left afterwards.
	if _, err := repo.ReleaseByName("FE2E-FE2E97", "", release.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ReleaseByName("FE2E-FE2E97", "", release.Add(2*time.Minute)); !errors.Is(err, ErrNothingToRelease) {
		t.Fatalf("second release: %v", err)
	}

	// Flash with release for a device in use (not assigned in the inventory, activated by the app).
	third := DeviceModel{Name: "FE2E-FE2E97", DeviceTypeModelID: deviceType.ID, AccountModelID: second.ID, ActivatedAt: ntime(release)}
	must(t, db.Omit("DeviceType").Create(&third).Error)
	rel := flash
	rel.Release = true
	d, err = repo.RecordFlash(rel, release.Add(time.Hour))
	must(t, err)
	if d.State != inventory.Flashed || d.InUse != nil {
		t.Fatalf("after flash with release: %+v", d)
	}
	both := flash
	both.Release, both.KeepAccount = true, true
	if _, err := repo.RecordFlash(both, release); !errors.Is(err, ErrReleaseAndKeep) {
		t.Fatalf("release and keep: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
