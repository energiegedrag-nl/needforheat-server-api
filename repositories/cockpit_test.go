package repositories

import (
	"testing"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cockpit"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1) // one in-memory database
	// CloudFeedModel has encrypted columns that SQLite stores as blobs; create the table by hand.
	if err := db.Exec("CREATE TABLE cloud_feed (account_id integer, cloud_feed_type_id integer, created_at datetime, updated_at datetime, deleted_at datetime, access_token blob, refresh_token blob, expiry datetime, auth_grant_token blob, activated_at datetime)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE cloud_feed_type (id integer primary key, name text, deleted_at datetime)").Error; err != nil {
		t.Fatal(err)
	}
	err = db.AutoMigrate(&AppModel{}, &DataSourceListModel{}, &CampaignModel{}, &DataSourceTypeModel{}, &AccountModel{},
		&UploadModel{}, &DeviceTypeModel{}, &DeviceModel{}, &DataSourceListItems{}, &EnergyQueryTypeModel{}, &EnergyQueryModel{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func ntime(t time.Time) *needforheat.Time {
	v := needforheat.Time(t)
	return &v
}

func TestCockpitCampaignStatus(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)

	app := AppModel{Name: "NeedForHeat", ProvisioningURLTemplate: "https://energiegedrag.nl/invite/<account_activation_token>"}
	must(t, db.Create(&app).Error)
	p1 := DeviceTypeModel{Name: "P1-Reader"}
	must(t, db.Create(&p1).Error)
	lrm := DeviceTypeModel{Name: "Living-Room-Module"}
	must(t, db.Create(&lrm).Error)
	dsP1 := DataSourceTypeModel{TypeInstanceID: p1.ID, TypeInstanceType: "device_type", NotificationThreshold: "P1D", UploadSchedule: "*/15 * * * *"}
	must(t, db.Create(&dsP1).Error)
	dsLrm := DataSourceTypeModel{TypeInstanceID: lrm.ID, TypeInstanceType: "device_type", NotificationThreshold: "PT12H"}
	must(t, db.Create(&dsLrm).Error)
	list := DataSourceListModel{Name: "Thuis"}
	must(t, db.Create(&list).Error)
	must(t, db.Create(&DataSourceListItems{DataSourceListModelID: list.ID, DataSourceTypeModelID: dsLrm.ID, Order: 2}).Error)
	must(t, db.Create(&DataSourceListItems{DataSourceListModelID: list.ID, DataSourceTypeModelID: dsP1.ID, Order: 1}).Error)

	campaign := CampaignModel{Name: "Winter 2026", AppModelID: app.ID, InfoURL: "https://energiegedrag.nl/winter", DataSourceListID: list.ID,
		StartTime: ntime(now.Add(-30 * 24 * time.Hour))}
	must(t, db.Omit("App").Create(&campaign).Error)

	invited := AccountModel{CampaignModelID: campaign.ID}
	must(t, db.Omit("Campaign").Create(&invited).Error)
	active := AccountModel{CampaignModelID: campaign.ID, ActivatedAt: ntime(now.Add(-10 * 24 * time.Hour))}
	must(t, db.Omit("Campaign").Create(&active).Error)

	// The active participant has a P1 reader that uploaded an hour ago and a living room module silent for two days.
	meter := DeviceModel{Name: "NFH-P1-AAAA", DeviceTypeModelID: p1.ID, AccountModelID: active.ID, ActivatedAt: ntime(now.Add(-9 * 24 * time.Hour))}
	must(t, db.Omit("DeviceType").Create(&meter).Error)
	room := DeviceModel{Name: "NFH-LRM-BBBB", DeviceTypeModelID: lrm.ID, AccountModelID: active.ID, ActivatedAt: ntime(now.Add(-9 * 24 * time.Hour))}
	must(t, db.Omit("DeviceType").Create(&room).Error)
	for _, u := range []UploadModel{
		{InstanceID: meter.ID, InstanceType: upload.Device, ServerTime: needforheat.Time(now.Add(-3 * time.Hour))},
		{InstanceID: meter.ID, InstanceType: upload.Device, ServerTime: needforheat.Time(now.Add(-time.Hour))},
		{InstanceID: room.ID, InstanceType: upload.Device, ServerTime: needforheat.Time(now.Add(-48 * time.Hour))},
	} {
		must(t, db.Create(&u).Error)
	}

	repo := NewCockpitRepository(db)
	status, err := repo.CampaignStatus(campaign.ID, now)
	must(t, err)

	if status.Campaign.State != cockpit.CampaignRunning || status.Campaign.Accounts != 2 || status.Campaign.Activated != 1 {
		t.Fatalf("campaign summary: %+v", status.Campaign)
	}
	if len(status.Sources) != 2 || status.Sources[0].Name != "P1-Reader" || status.Sources[1].Name != "Living-Room-Module" {
		t.Fatalf("sources not in list order: %+v", status.Sources)
	}
	if len(status.Accounts) != 2 {
		t.Fatalf("accounts: %+v", status.Accounts)
	}
	if a := status.Accounts[0]; a.ID != invited.ID || a.State != cockpit.AccountInvited || a.Sources[0].State != cockpit.SourceMissing {
		t.Errorf("invited account: %+v", a)
	}
	a := status.Accounts[1]
	if a.State != cockpit.AccountAttention {
		t.Errorf("active account state = %s, want attention", a.State)
	}
	if s := a.Sources[0]; s.State != cockpit.SourceOK || s.Instance != "NFH-P1-AAAA" || s.LatestUpload == nil || *s.LatestUpload != now.Add(-time.Hour).Unix() {
		t.Errorf("P1 source: %+v", s)
	}
	if s := a.Sources[1]; s.State != cockpit.SourceLate {
		t.Errorf("living room source: %+v", s)
	}
	if status.Totals[cockpit.AccountInvited] != 1 || status.Totals[cockpit.AccountAttention] != 1 {
		t.Errorf("totals: %v", status.Totals)
	}

	overview, err := repo.Overview(now)
	must(t, err)
	if len(overview.Campaigns) != 1 || overview.Campaigns[0].DataSourceList != "Thuis" || len(overview.Apps) != 1 || overview.DataSourceLists[0].Items != 2 {
		t.Errorf("overview: %+v", overview)
	}

	end := now.Add(90 * 24 * time.Hour).Unix()
	info := "https://energiegedrag.nl/winter-2026"
	must(t, repo.UpdateCampaign(campaign.ID, cockpit.CampaignUpdate{InfoURL: &info, EndTime: &end}))
	status, err = repo.CampaignStatus(campaign.ID, now)
	must(t, err)
	if status.Campaign.InfoURL != info || status.Campaign.EndTime == nil || *status.Campaign.EndTime != end {
		t.Errorf("after update: %+v", status.Campaign)
	}
	if err := repo.UpdateCampaign(9999, cockpit.CampaignUpdate{InfoURL: &info}); err == nil {
		t.Error("update of unknown campaign: want error")
	}
}
