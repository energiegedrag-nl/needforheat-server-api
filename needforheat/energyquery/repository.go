package energyquery

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
)

// A EnergyQueryRepository can load, store and delete EnergyQueries.
type EnergyQueryRepository interface {
	Find(energyQuery EnergyQuery) (EnergyQuery, error)
	GetProperties(energyQuery EnergyQuery) ([]property.Property, error)
	GetMeasurements(EnergyQuery EnergyQuery, filters map[string]string) ([]measurement.Measurement, error)
	GetAll() ([]EnergyQuery, error)
	Create(EnergyQuery) (EnergyQuery, error)
	Update(EnergyQuery) (EnergyQuery, error)
	Delete(EnergyQuery) error
	GetAllByAccount(accountID uint) ([]EnergyQuery, error)
}
