package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"gorm.io/gorm"
)

type EnergyQueryTypeRepository struct {
	db *gorm.DB
}

// Create a new EnergyQueryTypeRepository.
func NewEnergyQueryTypeRepository(db *gorm.DB) *EnergyQueryTypeRepository {
	return &EnergyQueryTypeRepository{
		db: db,
	}
}

// Database representation of a [energyquerytype.EnergyQueryType]
type EnergyQueryTypeModel struct {
	gorm.Model
	Name            string `gorm:"column:energy_query_variety"`
	Formula         string
	DataSourceTypes []DataSourceTypeModel `gorm:"polymorphic:TypeInstance;"`
}

// Set the name of the table in the database.
func (EnergyQueryTypeModel) TableName() string {
	return "energy_query_type"
}

// Create a EnergyQueryTypeModel from a [EnergyQueryType.EnergyQueryType].
func MakeEnergyQueryTypeModel(energyQueryType energyquerytype.EnergyQueryType) EnergyQueryTypeModel {
	return EnergyQueryTypeModel{
		Model:   gorm.Model{ID: energyQueryType.ID},
		Name:    energyQueryType.EnergyQueryVariety,
		Formula: energyQueryType.Formula,
	}
}

// Create a [energyquerytype.EnergyQueryType] from a EnergyQueryTypeModel.
func (m *EnergyQueryTypeModel) fromModel() energyquerytype.EnergyQueryType {
	return energyquerytype.EnergyQueryType{
		ID:                 m.Model.ID,
		EnergyQueryVariety: m.Name,
		Formula:            m.Formula,
	}
}

func (r *EnergyQueryTypeRepository) Find(energyQueryType energyquerytype.EnergyQueryType) (energyquerytype.EnergyQueryType, error) {
	EnergyQueryTypeModel := MakeEnergyQueryTypeModel(energyQueryType)
	err := r.db.Where(&EnergyQueryTypeModel).First(&EnergyQueryTypeModel).Error
	return EnergyQueryTypeModel.fromModel(), err
}

func (r *EnergyQueryTypeRepository) GetAll() ([]energyquerytype.EnergyQueryType, error) {
	var EnergyQueryTypes []energyquerytype.EnergyQueryType

	var EnergyQueryTypeModels []EnergyQueryTypeModel
	err := r.db.Find(&EnergyQueryTypeModels).Error
	if err != nil {
		return nil, err
	}

	for _, EnergyQueryTypeModel := range EnergyQueryTypeModels {
		EnergyQueryTypes = append(EnergyQueryTypes, EnergyQueryTypeModel.fromModel())
	}

	return EnergyQueryTypes, nil
}

func (r *EnergyQueryTypeRepository) Create(energyQueryType energyquerytype.EnergyQueryType) (energyquerytype.EnergyQueryType, error) {
	EnergyQueryTypeModel := MakeEnergyQueryTypeModel(energyQueryType)
	err := r.db.Create(&EnergyQueryTypeModel).Error
	return EnergyQueryTypeModel.fromModel(), err
}

func (r *EnergyQueryTypeRepository) Delete(energyQueryType energyquerytype.EnergyQueryType) error {
	EnergyQueryTypeModel := MakeEnergyQueryTypeModel(energyQueryType)
	return r.db.Delete(&EnergyQueryTypeModel).Error
}
