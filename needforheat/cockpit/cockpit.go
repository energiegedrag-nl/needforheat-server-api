// Package cockpit holds the read models of the NeedForHeat Cockpit, the web portal in which researchers set up
// campaigns, invite participants and watch whether data keeps coming in (design: docs/22 in the deployment repository).
//
// Times are Unix seconds, like elsewhere in the API.
package cockpit

import (
	"errors"
	"regexp"
	"strconv"
	"time"
)

// Campaign states, derived from start and end time.
const (
	CampaignPlanned = "planned" // no start time yet, or the start time is in the future
	CampaignRunning = "running"
	CampaignEnded   = "ended"
)

// Data source states for one participant.
const (
	SourceMissing = "missing" // not connected (no device, energy query or cloud feed of this type)
	SourceWaiting = "waiting" // connected, no upload yet, still within the notification threshold
	SourceOK      = "ok"      // uploads arrive within the notification threshold
	SourceLate    = "late"    // nothing received for longer than the notification threshold
)

// Participant (account) states.
const (
	AccountInvited    = "invited"    // invitation not used yet
	AccountActive     = "active"     // activated, all connected sources up to date
	AccountIncomplete = "incomplete" // activated, but not every data source is connected
	AccountAttention  = "attention"  // at least one data source is late
)

type App struct {
	ID                      uint   `json:"id"`
	Name                    string `json:"name"`
	ProvisioningURLTemplate string `json:"provisioning_url_template"`
	OauthRedirectURL        string `json:"oauth_redirect_url"`
}

type DataSourceList struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Items int    `json:"items"`
}

type Campaign struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	App              string `json:"app"`
	InfoURL          string `json:"info_url"`
	StartTime        *int64 `json:"start_time"`
	EndTime          *int64 `json:"end_time"`
	State            string `json:"state"`
	DataSourceListID uint   `json:"data_source_list_id"`
	DataSourceList   string `json:"data_source_list"`
	Accounts         int    `json:"accounts"`
	Activated        int    `json:"activated"`
}

type Overview struct {
	Apps            []App            `json:"apps"`
	Campaigns       []Campaign       `json:"campaigns"`
	DataSourceLists []DataSourceList `json:"data_source_lists"`
	GeneratedAt     int64            `json:"generated_at"`
}

// A Source is one item of the campaign's data source list (a column in the status table).
type Source struct {
	ID                    uint   `json:"id"`
	Category              string `json:"category"` // device_type, energy_query_type or cloud_feed_type
	Name                  string `json:"name"`
	UploadSchedule        string `json:"upload_schedule"`
	NotificationThreshold string `json:"notification_threshold"`
}

// A SourceStatus is the state of one data source for one participant.
type SourceStatus struct {
	SourceID     uint   `json:"source_id"`
	Instance     string `json:"instance,omitempty"` // device name, if any
	ActivatedAt  *int64 `json:"activated_at"`
	LatestUpload *int64 `json:"latest_upload"`
	State        string `json:"state"`
}

type AccountStatus struct {
	ID          uint           `json:"id"`
	ActivatedAt *int64         `json:"activated_at"`
	State       string         `json:"state"`
	Sources     []SourceStatus `json:"sources"`
}

type CampaignStatus struct {
	Campaign    Campaign        `json:"campaign"`
	Sources     []Source        `json:"sources"`
	Accounts    []AccountStatus `json:"accounts"`
	Totals      map[string]int  `json:"totals"` // number of participants per account state
	GeneratedAt int64           `json:"generated_at"`
}

// A CampaignUpdate changes a campaign; nil fields stay as they are.
type CampaignUpdate struct {
	InfoURL   *string `json:"info_url"`
	StartTime *int64  `json:"start_time"`
	EndTime   *int64  `json:"end_time"`
}

// An Invitation is a new participant account with its personal activation link.
type Invitation struct {
	AccountID     uint   `json:"account_id"`
	InvitationURL string `json:"invitation_url"`
}

// Unix returns t as Unix seconds, or nil.
func Unix(t *time.Time) *int64 {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Unix()
	return &s
}

// StateOfCampaign derives the campaign state from its start and end time.
func StateOfCampaign(start, end *time.Time, now time.Time) string {
	switch {
	case end != nil && !end.IsZero() && !now.Before(*end):
		return CampaignEnded
	case start == nil || start.IsZero() || now.Before(*start):
		return CampaignPlanned
	default:
		return CampaignRunning
	}
}

// StateOfSource classifies one data source of one participant, the same way the app warns about missed uploads:
// late when the latest upload (or, without uploads, the activation) is longer ago than the notification threshold.
// A threshold of 0 means no threshold.
func StateOfSource(activatedAt, latestUpload *time.Time, threshold time.Duration, now time.Time) string {
	if activatedAt == nil || activatedAt.IsZero() {
		return SourceMissing
	}
	since := *activatedAt
	if latestUpload != nil && !latestUpload.IsZero() {
		since = *latestUpload
	}
	if threshold > 0 && now.Sub(since) > threshold {
		return SourceLate
	}
	if latestUpload == nil || latestUpload.IsZero() {
		return SourceWaiting
	}
	return SourceOK
}

// StateOfAccount combines the states of the participant's data sources.
func StateOfAccount(activatedAt *time.Time, sources []SourceStatus) string {
	if activatedAt == nil || activatedAt.IsZero() {
		return AccountInvited
	}
	state := AccountActive
	for _, s := range sources {
		switch s.State {
		case SourceLate:
			return AccountAttention
		case SourceMissing:
			state = AccountIncomplete
		}
	}
	return state
}

var isoDuration = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// ErrDuration is returned for a notification threshold that is not an ISO 8601 duration.
var ErrDuration = errors.New("not an ISO 8601 duration")

// ParseDuration parses an ISO 8601 duration such as "P1D" or "PT12H" (a year counts as 365 days, a month as 30).
// An empty string gives 0.
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	m := isoDuration.FindStringSubmatch(s)
	if m == nil || s == "P" || s == "PT" {
		return 0, ErrDuration
	}
	units := []time.Duration{365 * 24 * time.Hour, 30 * 24 * time.Hour, 7 * 24 * time.Hour, 24 * time.Hour, time.Hour, time.Minute, time.Second}
	var d time.Duration
	for i, unit := range units {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return 0, ErrDuration
		}
		d += time.Duration(n) * unit
	}
	return d, nil
}
