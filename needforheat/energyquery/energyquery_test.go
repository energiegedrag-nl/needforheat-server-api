package energyquery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
)

// The app sends its uploads as JSON without a server time; the server sets it on creation.
func TestMakeEnergyQuerySetsTheServerTimeOfItsUploads(t *testing.T) {
	var uploads []upload.Upload
	body := `[{"device_time": 1764027817, "measurements": [
		{"property": {"name": "heat_tr_building_cond__W_K_1"}, "time": 1764027817, "value": "155"},
		{"property": {"name": "qv10__dm3_s_1_m_2"}, "time": 1764027817, "value": "0.5"}]}]`
	if err := json.Unmarshal([]byte(body), &uploads); err != nil {
		t.Fatal(err)
	}

	before := time.Now().Add(-time.Second)
	q := MakeEnergyQuery(energyquerytype.EnergyQueryType{}, 3, uploads)
	after := time.Now().Add(time.Second)

	if len(q.Uploads) != 1 {
		t.Fatalf("uploads: got %d, want 1", len(q.Uploads))
	}
	u := q.Uploads[0]
	st := time.Time(u.ServerTime)
	if st.Before(before) || st.After(after) {
		t.Errorf("server time %v not between %v and %v", st, before, after)
	}
	if !st.Equal(time.Time(*q.ActivatedAt)) {
		t.Errorf("server time %v differs from activated at %v", st, time.Time(*q.ActivatedAt))
	}
	if u.InstanceType != upload.EnergyQuery {
		t.Errorf("instance type: got %q, want %q", u.InstanceType, upload.EnergyQuery)
	}
	if u.Size != 2 {
		t.Errorf("size: got %d, want 2", u.Size)
	}
}
