package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/devicetype"
	"gorm.io/gorm"
)

type DeviceTypeRepository struct {
	db *gorm.DB
}

// Create a new DeviceTypeRepository.
func NewDeviceTypeRepository(db *gorm.DB) *DeviceTypeRepository {
	return &DeviceTypeRepository{
		db: db,
	}
}

// Database representation of a [devicetype.DeviceType]
type DeviceTypeModel struct {
	gorm.Model
	Name            string                `gorm:"unique;non null"`
	DataSourceTypes []DataSourceTypeModel `gorm:"polymorphic:TypeInstance;"`
}

// Set the name of the table in the database.
func (DeviceTypeModel) TableName() string {
	return "device_type"
}

// Create a DeviceTypeModel from a [devicetype.DeviceType].
func MakeDeviceTypeModel(deviceType devicetype.DeviceType) DeviceTypeModel {
	return DeviceTypeModel{
		Model: gorm.Model{ID: deviceType.ID},
		Name:  deviceType.Name,
	}
}

// Create a [devicetype.DeviceType] from a DeviceTypeModel.
func (m *DeviceTypeModel) fromModel() devicetype.DeviceType {
	return devicetype.DeviceType{
		ID:   m.Model.ID,
		Name: m.Name,
	}
}

func (r *DeviceTypeRepository) Find(deviceType devicetype.DeviceType) (devicetype.DeviceType, error) {
	deviceTypeModel := MakeDeviceTypeModel(deviceType)
	err := r.db.Where(&deviceTypeModel).First(&deviceTypeModel).Error
	return deviceTypeModel.fromModel(), err
}

func (r *DeviceTypeRepository) GetAll() ([]devicetype.DeviceType, error) {
	var deviceTypes []devicetype.DeviceType

	var deviceTypeModels []DeviceTypeModel
	err := r.db.Find(&deviceTypeModels).Error
	if err != nil {
		return nil, err
	}

	for _, deviceTypeModel := range deviceTypeModels {
		deviceTypes = append(deviceTypes, deviceTypeModel.fromModel())
	}

	return deviceTypes, nil
}

func (r *DeviceTypeRepository) Create(deviceType devicetype.DeviceType) (devicetype.DeviceType, error) {
	deviceTypeModel := MakeDeviceTypeModel(deviceType)
	err := r.db.Create(&deviceTypeModel).Error
	return deviceTypeModel.fromModel(), err
}

func (r *DeviceTypeRepository) Delete(deviceType devicetype.DeviceType) error {
	deviceTypeModel := MakeDeviceTypeModel(deviceType)
	return r.db.Delete(&deviceTypeModel).Error
}
