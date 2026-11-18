package campaign

import (
	"errors"
	"regexp"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/app"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcelist"
)

// A campaign is a timeframe where we gather measurements with a specific goal.
type Campaign struct {
	ID             uint                          `json:"id"`
	Name           string                        `json:"name"`
	App            app.App                       `json:"app"`
	InfoURL        string                        `json:"info_url"`
	StartTime      *needforheat.Time             `json:"start_time,omitempty"`
	EndTime        *needforheat.Time             `json:"end_time,omitempty"`
	DataSourceList datasourcelist.DataSourceList `json:"data_source_list"`
	// Default language of the campaign (BCP 47, e.g. "nl-NL"): for texts on devices, manuals and invitations.
	Language string `json:"language"`
}

// DefaultLanguage is the language of a campaign that does not set one.
const DefaultLanguage = "nl-NL"

var languagePattern = regexp.MustCompile(`^([a-z]{2,3})(?:[-_]([A-Za-z]{2}))?$`)

// ErrLanguage is returned for a language that is not of the form "nl" or "nl-NL".
var ErrLanguage = errors.New(`language must look like "nl-NL" or "en-US"`)

// NormalizeLanguage returns the language in the form "nl-NL" (or "nl"); empty gives DefaultLanguage.
func NormalizeLanguage(language string) (string, error) {
	if language == "" {
		return DefaultLanguage, nil
	}
	m := languagePattern.FindStringSubmatch(language)
	if m == nil {
		return "", ErrLanguage
	}
	if m[2] == "" {
		return m[1], nil
	}
	region := []byte(m[2])
	for i, c := range region {
		if c >= 'a' && c <= 'z' {
			region[i] = c - 'a' + 'A'
		}
	}
	return m[1] + "-" + string(region), nil
}

// Create a new Campaign.
func MakeCampaign(name string, app app.App, infoURL string, startTime, endTime *needforheat.Time, dataSourceList datasourcelist.DataSourceList) Campaign {
	return Campaign{
		Name:           name,
		App:            app,
		InfoURL:        infoURL,
		StartTime:      startTime,
		EndTime:        endTime,
		DataSourceList: dataSourceList,
	}
}
