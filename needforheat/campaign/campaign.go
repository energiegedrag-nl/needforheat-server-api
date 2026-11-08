package campaign

import (
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
