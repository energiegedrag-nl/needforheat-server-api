package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquery"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"gorm.io/gorm"
)

type EnergyQueryRepository struct {
	db *gorm.DB
}

// Create a new EnergyQueryRepository.
func NewEnergyQueryRepository(db *gorm.DB) *EnergyQueryRepository {
	return &EnergyQueryRepository{
		db: db,
	}
}

// Database representation of a [energyquery.EnergyQuery]
type EnergyQueryModel struct {
	gorm.Model
	EnergyQueryTypeModelID uint `gorm:"column:energy_query_type_id"`
	EnergyQueryType        EnergyQueryTypeModel
	AccountModelID         uint `gorm:"column:account_id"`
	ActivatedAt            *needforheat.Time
	Uploads                []UploadModel `gorm:"polymorphic:Instance;"`
}

// Set the name of the table in the database.
func (EnergyQueryModel) TableName() string {
	return "energy_query"
}

// Create a EnergyQueryModel from a [EnergyQuery.EnergyQuery].
func MakeEnergyQueryModel(energyQuery energyquery.EnergyQuery) EnergyQueryModel {
	var uploadModels []UploadModel

	for _, upload := range energyQuery.Uploads {
		uploadModels = append(uploadModels, MakeUploadModel(upload))
	}

	return EnergyQueryModel{
		Model:                  gorm.Model{ID: energyQuery.ID},
		EnergyQueryTypeModelID: energyQuery.EnergyQueryType.ID,
		EnergyQueryType:        MakeEnergyQueryTypeModel(energyQuery.EnergyQueryType),
		AccountModelID:         energyQuery.AccountID,
		ActivatedAt:            energyQuery.ActivatedAt,
		Uploads:                uploadModels,
	}
}

// Create a [energyquery.EnergyQuery] from a EnergyQueryModel.
func (m *EnergyQueryModel) fromModel() energyquery.EnergyQuery {
	var uploads []upload.Upload

	for _, uploadModel := range m.Uploads {
		uploads = append(uploads, uploadModel.fromModel())
	}

	return energyquery.EnergyQuery{
		ID:              m.Model.ID,
		EnergyQueryType: m.EnergyQueryType.fromModel(),
		AccountID:       m.AccountModelID,
		ActivatedAt:     m.ActivatedAt,
		Uploads:         uploads,
	}
}

func (r *EnergyQueryRepository) Find(energyQuery energyquery.EnergyQuery) (energyquery.EnergyQuery, error) {
	EnergyQueryModel := MakeEnergyQueryModel(energyQuery)
	err := r.db.Preload("EnergyQueryType").Preload("Uploads").Where(&EnergyQueryModel).First(&EnergyQueryModel).Error
	return EnergyQueryModel.fromModel(), err
}

func (r *EnergyQueryRepository) GetMeasurements(energyQuery energyquery.EnergyQuery, filters map[string]string) ([]measurement.Measurement, error) {
	// empty array of measurements
	var measurements []measurement.Measurement = make([]measurement.Measurement, 0)

	query := r.db.
		Model(&measurement.Measurement{}).
		Preload("Property").
		Joins("JOIN upload ON measurement.upload_id = upload.id").
		Joins("JOIN energy_query ON upload.instance_id = energy_query.id AND upload.instance_type = 'energy_query'").
		Where("energy_query.id = ?", energyQuery.ID)

	// apply filters
	for name, value := range filters {
		switch name {
		case "property":
			name = "property_id"
		case "start":
			name = "measurement.time >= ?"
		case "end":
			name = "measurement.time <= ?"
		}

		query = query.Where(name, value)
	}

	err := query.Find(&measurements).Error

	if err != nil {
		return nil, err
	}

	return measurements, nil
}

func (r *EnergyQueryRepository) GetProperties(energyQuery energyquery.EnergyQuery) ([]property.Property, error) {
	var properties []property.Property = make([]property.Property, 0)

	err := r.db.
		Table("energy_query").
		Select("DISTINCT property.id, property.name").
		Joins("JOIN upload ON energy_query.id = upload.instance_id AND upload.instance_type = 'energy_query'").
		Joins("JOIN measurement ON upload.id = measurement.upload_id").
		Joins("JOIN property ON property.id = measurement.property_id").
		Where("energy_query.id = ?", energyQuery.ID).
		Scan(&properties).
		Error

	if err != nil {
		return nil, err
	}

	return properties, nil
}

func (r *EnergyQueryRepository) GetAll() ([]energyquery.EnergyQuery, error) {
	var energyQueries []energyquery.EnergyQuery

	var EnergyQueryModels []EnergyQueryModel
	err := r.db.Preload("EnergyQueryType").Preload("Uploads").Find(&EnergyQueryModels).Error
	if err != nil {
		return nil, err
	}

	for _, EnergyQueryModel := range EnergyQueryModels {
		energyQueries = append(energyQueries, EnergyQueryModel.fromModel())
	}

	return energyQueries, nil
}

func (r *EnergyQueryRepository) GetAllByAccount(accountID uint) ([]energyquery.EnergyQuery, error) {
	var energyQueries []energyquery.EnergyQuery
	var EnergyQueryModels []EnergyQueryModel

	err := r.db.Where("account_id = ?", accountID).Preload("EnergyQueryType").Find(&EnergyQueryModels).Error
	if err != nil {
		return nil, err
	}

	for _, EnergyQueryModel := range EnergyQueryModels {
		energyQueries = append(energyQueries, EnergyQueryModel.fromModel())
	}

	return energyQueries, nil
}

func (r *EnergyQueryRepository) Create(energyQuery energyquery.EnergyQuery) (energyquery.EnergyQuery, error) {
	EnergyQueryModel := MakeEnergyQueryModel(energyQuery)
	err := r.db.Preload("").Create(&EnergyQueryModel).Error
	return EnergyQueryModel.fromModel(), err
}

func (r *EnergyQueryRepository) Update(energyQuery energyquery.EnergyQuery) (energyquery.EnergyQuery, error) {
	EnergyQueryModel := MakeEnergyQueryModel(energyQuery)
	err := r.db.Model(&EnergyQueryModel).Updates(EnergyQueryModel).Error
	return EnergyQueryModel.fromModel(), err
}

func (r *EnergyQueryRepository) Delete(energyQuery energyquery.EnergyQuery) error {
	EnergyQueryModel := MakeEnergyQueryModel(energyQuery)
	return r.db.Delete(&EnergyQueryModel).Error
}
