package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

// Create a new DeviceRepository.
func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{
		db: db,
	}
}

// Database representation of a [device.Device]
type DeviceModel struct {
	gorm.Model
	Name                 string `gorm:"unique;not null"`
	DeviceTypeModelID    uint   `gorm:"column:device_type_id"`
	DeviceType           DeviceTypeModel
	AccountModelID       uint `gorm:"column:account_id"`
	ActivationSecretHash string
	ActivatedAt          *needforheat.Time
	Uploads              []UploadModel `gorm:"polymorphic:Instance;"`
}

// Set the name of the table in the database.
func (DeviceModel) TableName() string {
	return "device"
}

// Create a DeviceModel from a [device.Device].
func MakeDeviceModel(device device.Device) DeviceModel {
	var uploadModels []UploadModel

	for _, upload := range device.Uploads {
		uploadModels = append(uploadModels, MakeUploadModel(upload))
	}

	return DeviceModel{
		Model:                gorm.Model{ID: device.ID},
		Name:                 device.Name,
		DeviceTypeModelID:    device.DeviceType.ID,
		DeviceType:           MakeDeviceTypeModel(device.DeviceType),
		AccountModelID:       device.AccountID,
		ActivationSecretHash: device.ActivationSecretHash,
		ActivatedAt:          device.ActivatedAt,
		Uploads:              uploadModels,
	}
}

// Create a [device.Device] from a DeviceModel.
func (m *DeviceModel) fromModel() device.Device {
	var uploads []upload.Upload

	for _, uploadModel := range m.Uploads {
		uploads = append(uploads, uploadModel.fromModel())
	}

	return device.Device{
		ID:                   m.Model.ID,
		Name:                 m.Name,
		DeviceType:           m.DeviceType.fromModel(),
		AccountID:            m.AccountModelID,
		ActivationSecretHash: m.ActivationSecretHash,
		ActivatedAt:          m.ActivatedAt,
		Uploads:              uploads,
	}
}

func (r *DeviceRepository) Find(device device.Device) (device.Device, error) {
	deviceModel := MakeDeviceModel(device)
	err := r.db.Preload("DeviceType").Preload("Uploads").Where(&deviceModel).First(&deviceModel).Error
	return deviceModel.fromModel(), err
}

func (r *DeviceRepository) FindCloudFeedAuthCreationTimeFromDeviceID(deviceID uint) (*needforheat.Time, error) {
	result := struct {
		CreatedAt needforheat.Time
	}{}

	err := r.db.
		Table("device").
		Select("cloud_feed.created_at").
		Joins("JOIN device_type ON device.device_type_id = device_type.id").
		Joins("JOIN cloud_feed_type ON device_type.name = cloud_feed_type.name").
		Joins("JOIN account ON device.account_id = account.id").
		Joins("JOIN cloud_feed ON account.id = cloud_feed.account_id").
		Where("device.id = ?", deviceID).
		First(&result).
		Error

	if err != nil {
		return nil, err
	}

	return &result.CreatedAt, nil
}

func (r *DeviceRepository) GetMeasurements(device device.Device, filters map[string]string) ([]measurement.Measurement, error) {
	// empty array of measurements
	var measurements []measurement.Measurement = make([]measurement.Measurement, 0)

	query := r.db.
		Model(&measurement.Measurement{}).
		Preload("Property").
		Joins("JOIN upload ON measurement.upload_id = upload.id AND upload.instance_type = 'device'").
		Joins("JOIN device ON upload.instance_id = device.id").
		Where("device.id = ?", device.ID)

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

func (r *DeviceRepository) GetProperties(device device.Device) ([]property.Property, error) {
	var properties []property.Property = make([]property.Property, 0)

	err := r.db.
		Table("device").
		Select("DISTINCT property.id, property.name").
		Joins("JOIN upload ON device.id = upload.instance_id AND upload.instance_type = 'device'").
		Joins("JOIN measurement ON upload.id = measurement.upload_id").
		Joins("JOIN property ON property.id = measurement.property_id").
		Where("device.id = ?", device.ID).
		Scan(&properties).
		Error

	if err != nil {
		return nil, err
	}

	return properties, nil
}

func (r *DeviceRepository) GetAll() ([]device.Device, error) {
	var devices []device.Device

	var deviceModels []DeviceModel
	err := r.db.Preload("DeviceType").Preload("Uploads").Find(&deviceModels).Error
	if err != nil {
		return nil, err
	}

	for _, deviceModel := range deviceModels {
		devices = append(devices, deviceModel.fromModel())
	}

	return devices, nil
}

func (r *DeviceRepository) GetAllByAccount(accountID uint) ([]device.Device, error) {
	var devices []device.Device
	var deviceModels []DeviceModel

	err := r.db.Where("account_id = ?", accountID).Preload("DeviceType").Find(&deviceModels).Error
	if err != nil {
		return nil, err
	}

	for _, deviceModel := range deviceModels {
		devices = append(devices, deviceModel.fromModel())
	}

	return devices, nil
}

func (r *DeviceRepository) Create(device device.Device) (device.Device, error) {
	deviceModel := MakeDeviceModel(device)
	err := r.db.Preload("").Create(&deviceModel).Error
	return deviceModel.fromModel(), err
}

func (r *DeviceRepository) Update(device device.Device) (device.Device, error) {
	deviceModel := MakeDeviceModel(device)
	err := r.db.Model(&deviceModel).Updates(deviceModel).Error
	return deviceModel.fromModel(), err
}

func (r *DeviceRepository) Delete(device device.Device) error {
	deviceModel := MakeDeviceModel(device)
	return r.db.Delete(&deviceModel).Error
}
