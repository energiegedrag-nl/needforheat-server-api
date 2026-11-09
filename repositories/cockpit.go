package repositories

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cockpit"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"gorm.io/gorm"
)

// CockpitRepository reads the overviews and status tables of the NeedForHeat Cockpit straight from the database.
// It only reads, except for UpdateCampaign.
type CockpitRepository struct {
	db *gorm.DB
}

func NewCockpitRepository(db *gorm.DB) *CockpitRepository {
	return &CockpitRepository{db: db}
}

func timePtr(t *needforheat.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := time.Time(*t)
	if v.IsZero() {
		return nil
	}
	return &v
}

type campaignCount struct {
	CampaignID uint
	Total      int
	Activated  int
}

func (r *CockpitRepository) campaignSummary(m CampaignModel, counts map[uint]campaignCount, lists map[uint]string, now time.Time) cockpit.Campaign {
	start, end := timePtr(m.StartTime), timePtr(m.EndTime)
	c := counts[m.ID]
	return cockpit.Campaign{
		ID:               m.ID,
		Name:             m.Name,
		App:              m.App.Name,
		InfoURL:          m.InfoURL,
		StartTime:        cockpit.Unix(start),
		EndTime:          cockpit.Unix(end),
		State:            cockpit.StateOfCampaign(start, end, now),
		DataSourceListID: m.DataSourceListID,
		DataSourceList:   lists[m.DataSourceListID],
		Accounts:         c.Total,
		Activated:        c.Activated,
	}
}

func (r *CockpitRepository) counts(campaignIDs ...uint) (map[uint]campaignCount, error) {
	var rows []campaignCount
	q := r.db.Model(&AccountModel{}).
		Select("campaign_id, COUNT(*) AS total, COUNT(activated_at) AS activated").
		Group("campaign_id")
	if len(campaignIDs) > 0 {
		q = q.Where("campaign_id IN ?", campaignIDs)
	}
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[uint]campaignCount{}
	for _, row := range rows {
		out[row.CampaignID] = row
	}
	return out, nil
}

func (r *CockpitRepository) listNames() (map[uint]string, []cockpit.DataSourceList, error) {
	var lists []DataSourceListModel
	if err := r.db.Preload("Items").Order("id").Find(&lists).Error; err != nil {
		return nil, nil, err
	}
	names := map[uint]string{}
	out := []cockpit.DataSourceList{}
	for _, l := range lists {
		names[l.ID] = l.Name
		out = append(out, cockpit.DataSourceList{ID: l.ID, Name: l.Name, Items: len(l.Items)})
	}
	return names, out, nil
}

// Overview lists apps, campaigns (with their number of participants) and data source lists.
func (r *CockpitRepository) Overview(now time.Time) (cockpit.Overview, error) {
	out := cockpit.Overview{Apps: []cockpit.App{}, Campaigns: []cockpit.Campaign{}, GeneratedAt: now.Unix()}

	var apps []AppModel
	if err := r.db.Order("id").Find(&apps).Error; err != nil {
		return out, err
	}
	for _, a := range apps {
		out.Apps = append(out.Apps, cockpit.App{ID: a.ID, Name: a.Name, ProvisioningURLTemplate: a.ProvisioningURLTemplate, OauthRedirectURL: a.OauthRedirectURL})
	}

	names, lists, err := r.listNames()
	if err != nil {
		return out, err
	}
	out.DataSourceLists = lists

	counts, err := r.counts()
	if err != nil {
		return out, err
	}

	var campaigns []CampaignModel
	if err := r.db.Preload("App").Order("id DESC").Find(&campaigns).Error; err != nil {
		return out, err
	}
	for _, c := range campaigns {
		out.Campaigns = append(out.Campaigns, r.campaignSummary(c, counts, names, now))
	}
	return out, nil
}

type sourceRow struct {
	DataSourceTypeID      uint
	TypeInstanceID        uint
	TypeInstanceType      string
	UploadSchedule        string
	NotificationThreshold string
}

func (r *CockpitRepository) nameMap(table, column string) (map[uint]string, error) {
	var rows []struct {
		ID   uint
		Name string
	}
	if err := r.db.Table(table).Select("id, " + column + " AS name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := map[uint]string{}
	for _, row := range rows {
		out[row.ID] = row.Name
	}
	return out, nil
}

// latestUpload returns the server time of the newest upload of a device or energy query (uses idx_upload_instance).
func (r *CockpitRepository) latestUpload(kind upload.InstanceType, id uint) (*time.Time, error) {
	var rows []UploadModel
	err := r.db.Select("server_time").
		Where("instance_type = ? AND instance_id = ?", kind, id).
		Order("server_time DESC").Limit(1).
		Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	t := time.Time(rows[0].ServerTime)
	if t.IsZero() {
		return nil, nil
	}
	return &t, nil
}

// CampaignStatus gives, per participant and per data source of the campaign, the latest upload and its state.
func (r *CockpitRepository) CampaignStatus(campaignID uint, now time.Time) (cockpit.CampaignStatus, error) {
	var out cockpit.CampaignStatus

	var c CampaignModel
	if err := r.db.Preload("App").First(&c, campaignID).Error; err != nil {
		return out, err
	}
	names, _, err := r.listNames()
	if err != nil {
		return out, err
	}
	counts, err := r.counts(c.ID)
	if err != nil {
		return out, err
	}
	out.Campaign = r.campaignSummary(c, counts, names, now)
	out.GeneratedAt = now.Unix()

	// The campaign's data sources, in list order.
	var rows []sourceRow
	err = r.db.Table("data_source_list_items AS i").
		Select("t.id AS data_source_type_id, t.type_instance_id, t.type_instance_type, t.upload_schedule, t.notification_threshold").
		Joins("JOIN data_source_type t ON t.id = i.data_source_type_model_id AND t.deleted_at IS NULL").
		Where("i.data_source_list_model_id = ?", c.DataSourceListID).
		Order("i.`order`, i.id").
		Scan(&rows).Error
	if err != nil {
		return out, err
	}
	deviceTypes, err := r.nameMap("device_type", "name")
	if err != nil {
		return out, err
	}
	queryTypes, err := r.nameMap("energy_query_type", "energy_query_variety")
	if err != nil {
		return out, err
	}
	feedTypes, err := r.nameMap("cloud_feed_type", "name")
	if err != nil {
		return out, err
	}
	thresholds := make([]time.Duration, len(rows))
	out.Sources = []cockpit.Source{}
	for i, s := range rows {
		name := ""
		switch s.TypeInstanceType {
		case "device_type":
			name = deviceTypes[s.TypeInstanceID]
		case "energy_query_type":
			name = queryTypes[s.TypeInstanceID]
		case "cloud_feed_type":
			name = feedTypes[s.TypeInstanceID]
		}
		thresholds[i], _ = cockpit.ParseDuration(s.NotificationThreshold) // an invalid threshold counts as none
		out.Sources = append(out.Sources, cockpit.Source{
			ID: s.DataSourceTypeID, Category: s.TypeInstanceType, Name: name,
			UploadSchedule: s.UploadSchedule, NotificationThreshold: s.NotificationThreshold,
		})
	}

	// Participants and what they connected.
	var accounts []AccountModel
	if err := r.db.Where("campaign_id = ?", c.ID).Order("id").Find(&accounts).Error; err != nil {
		return out, err
	}
	inCampaign := r.db.Model(&AccountModel{}).Select("id").Where("campaign_id = ?", c.ID)
	var devices []DeviceModel
	if err := r.db.Where("account_id IN (?)", inCampaign).Order("id").Find(&devices).Error; err != nil {
		return out, err
	}
	var queries []EnergyQueryModel
	if err := r.db.Where("account_id IN (?)", inCampaign).Order("id").Find(&queries).Error; err != nil {
		return out, err
	}
	var feeds []CloudFeedModel
	if err := r.db.Where("account_id IN (?)", inCampaign).Find(&feeds).Error; err != nil {
		return out, err
	}

	out.Totals = map[string]int{
		cockpit.AccountInvited: 0, cockpit.AccountActive: 0, cockpit.AccountIncomplete: 0, cockpit.AccountAttention: 0,
	}
	out.Accounts = []cockpit.AccountStatus{}
	for _, a := range accounts {
		status := cockpit.AccountStatus{ID: a.ID, ActivatedAt: cockpit.Unix(timePtr(a.ActivatedAt)), Sources: []cockpit.SourceStatus{}}
		for i, s := range rows {
			st := cockpit.SourceStatus{SourceID: s.DataSourceTypeID}
			var activated, latest *time.Time
			switch s.TypeInstanceType {
			case "device_type":
				// A participant may have more than one device of a type (a replacement): show the one with the
				// newest upload, or else the newest device.
				for _, d := range devices {
					if d.AccountModelID != a.ID || d.DeviceTypeModelID != s.TypeInstanceID {
						continue
					}
					l, err := r.latestUpload(upload.Device, d.ID)
					if err != nil {
						return out, err
					}
					newer := l != nil && (latest == nil || l.After(*latest))
					if st.Instance == "" || newer || (l == nil && latest == nil) {
						st.Instance, activated, latest = d.Name, timePtr(d.ActivatedAt), l
					}
				}
			case "energy_query_type":
				for _, q := range queries {
					if q.AccountModelID != a.ID || q.EnergyQueryTypeModelID != s.TypeInstanceID {
						continue
					}
					l, err := r.latestUpload(upload.EnergyQuery, q.ID)
					if err != nil {
						return out, err
					}
					activated, latest = timePtr(q.ActivatedAt), l
				}
			case "cloud_feed_type":
				// Cloud feed data is stored as uploads of a device whose device type has the cloud feed's name.
				for _, f := range feeds {
					if f.AccountID != a.ID || f.CloudFeedTypeID != s.TypeInstanceID {
						continue
					}
					activated = timePtr(f.ActivatedAt)
					for _, d := range devices {
						if d.AccountModelID == a.ID && deviceTypes[d.DeviceTypeModelID] == feedTypes[f.CloudFeedTypeID] {
							l, err := r.latestUpload(upload.Device, d.ID)
							if err != nil {
								return out, err
							}
							if l != nil && (latest == nil || l.After(*latest)) {
								latest = l
							}
						}
					}
				}
			}
			st.ActivatedAt, st.LatestUpload = cockpit.Unix(activated), cockpit.Unix(latest)
			st.State = cockpit.StateOfSource(activated, latest, thresholds[i], now)
			status.Sources = append(status.Sources, st)
		}
		status.State = cockpit.StateOfAccount(timePtr(a.ActivatedAt), status.Sources)
		out.Totals[status.State]++
		out.Accounts = append(out.Accounts, status)
	}
	return out, nil
}

// UpdateCampaign changes the info URL, start time or end time of a campaign.
func (r *CockpitRepository) UpdateCampaign(campaignID uint, u cockpit.CampaignUpdate) error {
	changes := map[string]any{}
	if u.InfoURL != nil {
		changes["info_url"] = *u.InfoURL
	}
	if u.StartTime != nil {
		changes["start_time"] = needforheat.Time(time.Unix(*u.StartTime, 0).UTC())
	}
	if u.EndTime != nil {
		changes["end_time"] = needforheat.Time(time.Unix(*u.EndTime, 0).UTC())
	}
	if err := r.db.First(&CampaignModel{}, campaignID).Error; err != nil {
		return err
	}
	if len(changes) == 0 {
		return nil
	}
	return r.db.Model(&CampaignModel{}).Where("id = ?", campaignID).Updates(changes).Error
}
