package repositories

import (
	"testing"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/propertylabel"
)

func TestPropertyLabelsDefaultsAndCampaignOverrides(t *testing.T) {
	db := testDB(t)
	if err := db.AutoMigrate(&PropertyLabelModel{}); err != nil {
		t.Fatal(err)
	}
	r := NewPropertyLabelRepository(db)
	const h = "heat_tr_building_cond__W_K_1"
	const c = "th_mass_building__Wh_K_1"
	err := r.Set([]propertylabel.PropertyLabel{
		{PropertyName: h, Language: "nl-NL", Label: "Warmteverlies"},
		{PropertyName: h, Language: "en-US", Label: "Heat loss"},
		{PropertyName: c, Language: "nl-NL", Label: "Warmtecapaciteit"},
		{PropertyName: h, CampaignID: 7, Language: "nl-NL", Label: "Warmteverliescoëfficiënt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	check := func(campaignID uint, language string, want map[string]string) {
		t.Helper()
		got, err := r.Resolve(campaignID, language)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(want) {
			t.Errorf("campaign %d %s: got %v, want %v", campaignID, language, got, want)
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("campaign %d %s %s: got %q, want %q", campaignID, language, k, got[k], v)
			}
		}
	}
	check(7, "nl-NL", map[string]string{h: "Warmteverliescoëfficiënt", c: "Warmtecapaciteit"})
	check(8, "nl-NL", map[string]string{h: "Warmteverlies", c: "Warmtecapaciteit"})
	check(7, "en-US", map[string]string{h: "Heat loss"})

	// setting again replaces; an empty label removes the campaign's own, so the default shows again
	if err := r.Set([]propertylabel.PropertyLabel{
		{PropertyName: c, Language: "nl-NL", Label: "Warmtecapaciteit (C)"},
		{PropertyName: h, CampaignID: 7, Language: "nl-NL", Label: ""},
	}); err != nil {
		t.Fatal(err)
	}
	check(7, "nl-NL", map[string]string{h: "Warmteverlies", c: "Warmtecapaciteit (C)"})
}
