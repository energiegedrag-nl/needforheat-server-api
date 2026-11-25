package energyquery

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
)

// An EnergyQuery is collects measurements in a subject's account.
type EnergyQuery struct {
	ID              uint                            `json:"id"`
	EnergyQueryType energyquerytype.EnergyQueryType `json:"energy_query_type"`
	AccountID       uint                            `json:"account_id"` // This can be removed if a device uses JWT's too.
	ActivatedAt     *needforheat.Time               `json:"activated_at"`
	Uploads         []upload.Upload                 `json:"uploads,omitempty"`
}

// Create a new EnergyQuery, this should be with uploads
//
// The uploads come from the app as JSON, so the server sets what is its own: the server time
// (the same moment as the activation), the instance type and the number of measurements.
func MakeEnergyQuery(energyQueryType energyquerytype.EnergyQueryType, accountID uint, uploads []upload.Upload) EnergyQuery {
	now := time.Now().Unix()
	activatedAt := needforheat.Time(time.Unix(now, 0))
	for i := range uploads {
		uploads[i].ServerTime = needforheat.Time(time.Unix(now, 0).UTC())
		uploads[i].InstanceType = upload.EnergyQuery
		uploads[i].Size = len(uploads[i].Measurements)
	}
	return EnergyQuery{
		EnergyQueryType: energyQueryType,
		AccountID:       accountID,
		ActivatedAt:     &activatedAt,
		Uploads:         uploads,
	}
}
