package repositories

import (
	"errors"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcetype"
	"gorm.io/gorm"
)

type DataSourceTypeRepository struct {
	db *gorm.DB
}

func NewDataSourceTypeRepository(db *gorm.DB) *DataSourceTypeRepository {
	return &DataSourceTypeRepository{
		db: db,
	}
}

// Database representation of a [datasourcetype.DataSourceType].
type DataSourceTypeModel struct {
	gorm.Model
	TypeInstanceID        uint
	TypeInstanceType      string //Category in API
	Order                 uint   `gorm:"-"` //Used for returning the Order with DataSourceList
	InstallationManualURL string
	FAQURL                string
	InfoURL               string
	Precedes              []DataSourceTypeModel `gorm:"many2many:data_source_precedence;"`
	UploadSchedule        string                `gorm:"type:text"`
	MeasurementSchedule   string                `gorm:"type:text"`
	NotificationThreshold string
}

// Set the name of the table in the database.
func (DataSourceTypeModel) TableName() string {
	return "data_source_type"
}

// Create a new DataSourceTypeModel from a [datasourcetype.ShoppinglistItem]
func MakeDataSourceTypeModel(datasourcetype datasourcetype.DataSourceType) DataSourceTypeModel {
	var shoppingListItemModels []DataSourceTypeModel
	for _, item := range datasourcetype.Precedes {
		shoppingListItemModels = append(shoppingListItemModels, MakeDataSourceTypeModel(item))
	}

	return DataSourceTypeModel{
		Model:                 gorm.Model{ID: datasourcetype.ID},
		TypeInstanceID:        datasourcetype.TypeInstanceID,
		TypeInstanceType:      string(datasourcetype.Category),
		Order:                 datasourcetype.Order,
		InstallationManualURL: datasourcetype.InstallationManualURL,
		FAQURL:                datasourcetype.FAQURL,
		InfoURL:               datasourcetype.InfoURL,
		Precedes:              shoppingListItemModels,
		UploadSchedule:        datasourcetype.UploadSchedule,
		MeasurementSchedule:   datasourcetype.MeasurementSchedule,
		NotificationThreshold: datasourcetype.NotificationThreshold,
	}
}

// Create a [datasourcetype.DataSourceType] from a DataSourceTypeModel
func (m *DataSourceTypeModel) fromModel(db *gorm.DB) datasourcetype.DataSourceType {
	var items []datasourcetype.DataSourceType
	for _, shoppingListItemModel := range m.Precedes {
		items = append(items, shoppingListItemModel.fromModel(db))
	}

	var category = StringToCategory(m.TypeInstanceType)

	return datasourcetype.DataSourceType{
		ID:                    m.Model.ID,
		TypeInstanceID:        m.TypeInstanceID,
		Item:                  getItemFromDB(category, m.TypeInstanceID, db),
		Category:              category,
		Order:                 m.Order,
		InstallationManualURL: m.InstallationManualURL,
		FAQURL:                m.FAQURL,
		InfoURL:               m.InfoURL,
		Precedes:              items,
		UploadSchedule:        m.UploadSchedule,
		MeasurementSchedule:   m.MeasurementSchedule,
		NotificationThreshold: m.NotificationThreshold,
	}
}

func (r *DataSourceTypeRepository) Create(datasourcetype datasourcetype.DataSourceType) (datasourcetype.DataSourceType, error) {
	shoppingListItemModel := MakeDataSourceTypeModel(datasourcetype)
	err := r.db.Create(&shoppingListItemModel).Error
	return shoppingListItemModel.fromModel(r.db), err
}

func (r *DataSourceTypeRepository) Delete(datasourcetype datasourcetype.DataSourceType) error {
	shoppingListItemModel := MakeDataSourceTypeModel(datasourcetype)
	return r.db.Create(&shoppingListItemModel).Error
}

func (r *DataSourceTypeRepository) Find(shoppingListItem datasourcetype.DataSourceType) (datasourcetype.DataSourceType, error) {
	shoppingListItemModel := MakeDataSourceTypeModel(shoppingListItem)
	err := r.db.Where(&shoppingListItemModel).First(&shoppingListItemModel).Error
	return shoppingListItemModel.fromModel(r.db), err
}

func (r *DataSourceTypeRepository) GetAll() ([]datasourcetype.DataSourceType, error) {
	var shoppingListItems []datasourcetype.DataSourceType

	var shoppingListItemModels []DataSourceTypeModel
	err := r.db.Find(&shoppingListItemModels).Error
	if err != nil {
		return nil, err
	}

	for _, shoppingListItemModel := range shoppingListItemModels {
		shoppingListItems = append(shoppingListItems, shoppingListItemModel.fromModel(r.db))
	}

	return shoppingListItems, nil
}

// Check if we did not make a loop that can softlock the app
func (s *DataSourceTypeModel) AfterSave(tx *gorm.DB) (err error) {
	var emptySlice []uint
	if s.CheckforCircular(s, emptySlice) {
		if err := tx.Rollback().Error; err != nil {
			return err
		}
		return errors.New("circular reference detected, transaction rolled back")
	}
	return nil
}

func (s *DataSourceTypeModel) CheckforCircular(item *DataSourceTypeModel, previousIDs []uint) bool {
	previousIDs = append(previousIDs, item.ID)
	for _, elem := range item.Precedes {
		for _, ID := range previousIDs {
			if elem.ID == ID || s.CheckforCircular(&elem, previousIDs) {
				return true
			}
		}
	}
	return false
}

func StringToCategory(category string) datasourcetype.Category {
	switch category {
	case "device_type":
		return datasourcetype.DeviceType
	case "cloud_feed_type":
		return datasourcetype.CloudFeedType
	case "energy_query_type":
		return datasourcetype.EnergyQueryType
	default:
		return ""
	}
}

// Function to retrieve an item based on category and type instance ID
func getItemFromDB(category datasourcetype.Category, typeInstanceID uint, db *gorm.DB) interface{} {
	var item interface{}

	switch category {
	case datasourcetype.DeviceType:
		var deviceType DeviceTypeModel
		db.Table("device_type").Where("id = ?", typeInstanceID).First(&deviceType)
		item = deviceType
	case datasourcetype.CloudFeedType:
		var cloudFeedType CloudFeedTypeModel
		db.Table("cloud_feed_type").Where("id = ?", typeInstanceID).First(&cloudFeedType)
		item = cloudFeedType
	case datasourcetype.EnergyQueryType:
		var energyQueryType EnergyQueryTypeModel
		db.Table("energy_query_type").Where("id = ?", typeInstanceID).First(&energyQueryType)
		item = energyQueryType
	default:
		return nil
	}

	return item
}
