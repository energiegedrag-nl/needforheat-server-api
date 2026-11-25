package repositories

import (
	"errors"
	"testing"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquery"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"gorm.io/gorm"
)

// An account with an older weather location and a newer heat signature: asking for the heat
// signature by the name of its type must give the heat signature, not the oldest query.
func TestFindEnergyQueryByTheNameOfItsType(t *testing.T) {
	db := testDB(t)
	weather := EnergyQueryTypeModel{Name: "weather-interpolation-location"}
	building := EnergyQueryTypeModel{Name: "building-profile"}
	for _, m := range []*EnergyQueryTypeModel{&weather, &building} {
		if err := db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	first := EnergyQueryModel{EnergyQueryTypeModelID: weather.ID, AccountModelID: 3}
	second := EnergyQueryModel{EnergyQueryTypeModelID: building.ID, AccountModelID: 3}
	other := EnergyQueryModel{EnergyQueryTypeModelID: building.ID, AccountModelID: 4}
	for _, m := range []*EnergyQueryModel{&first, &second, &other} {
		if err := db.Create(m).Error; err != nil {
			t.Fatal(err)
		}
	}
	r := NewEnergyQueryRepository(db)
	find := func(variety string, account uint) (energyquery.EnergyQuery, error) {
		return r.Find(energyquery.EnergyQuery{
			EnergyQueryType: energyquerytype.EnergyQueryType{EnergyQueryVariety: variety},
			AccountID:       account,
		})
	}

	got, err := find("building-profile", 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != second.ID || got.EnergyQueryType.EnergyQueryVariety != "building-profile" {
		t.Errorf("building-profile of account 3: got query %d (%s), want %d", got.ID, got.EnergyQueryType.EnergyQueryVariety, second.ID)
	}
	if got, err := find("weather-interpolation-location", 3); err != nil || got.ID != first.ID {
		t.Errorf("weather of account 3: got query %d (%v), want %d", got.ID, err, first.ID)
	}
	if _, err := find("weather-interpolation-location", 4); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("weather of account 4: want not found, got %v", err)
	}
	if _, err := find("no-such-type", 3); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("unknown type: want not found, got %v", err)
	}
	// by id, as before
	if got, err := r.Find(energyquery.EnergyQuery{ID: second.ID}); err != nil || got.AccountID != 3 {
		t.Errorf("by id: got %+v (%v)", got, err)
	}
}
