package repositories

import (
	"github.com/energietransitie/needforheat-server-api/needforheat"
	"github.com/energietransitie/needforheat-server-api/needforheat/measurement"
	"github.com/energietransitie/needforheat-server-api/needforheat/upload"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UploadRepository struct {
	db *gorm.DB
}

// Create a new UploadRepository.
func NewUploadRepository(db *gorm.DB) *UploadRepository {
	return &UploadRepository{
		db: db,
	}
}

// Database representation of a [upload.Upload]
type UploadModel struct {
	gorm.Model
	// Composite index for "uploads of this device/energy query, newest first" (app status checks).
	InstanceID   uint                `gorm:"column:instance_id;index:idx_upload_instance,priority:2"`
	InstanceType upload.InstanceType `gorm:"size:32;default:device;index:idx_upload_instance,priority:1"`
	ServerTime   needforheat.Time    `gorm:"index:idx_upload_instance,priority:3"`
	DeviceTime   needforheat.Time
	Size         int
	Measurements []MeasurementModel
}

// Set the name of the table in the database.
func (UploadModel) TableName() string {
	return "upload"
}

// Create an UploadModel from a [upload.Upload].
func MakeUploadModel(upload upload.Upload) UploadModel {
	var measurementModels []MeasurementModel

	for _, measurement := range upload.Measurements {
		measurementModels = append(measurementModels, MakeMeasurementModel(measurement))
	}

	return UploadModel{
		Model:        gorm.Model{ID: upload.ID},
		InstanceID:   upload.InstanceID,
		InstanceType: upload.InstanceType,
		ServerTime:   upload.ServerTime,
		DeviceTime:   upload.DeviceTime,
		Size:         upload.Size,
		Measurements: measurementModels,
	}
}

// Create a [upload.Upload] from an UploadModel.
func (m *UploadModel) fromModel() upload.Upload {
	var measurements []measurement.Measurement

	for _, measurementModel := range m.Measurements {
		measurements = append(measurements, measurementModel.fromModel())
	}

	return upload.Upload{
		ID:           m.Model.ID,
		InstanceID:   m.InstanceID,
		InstanceType: StringToType(string(m.InstanceType)),
		ServerTime:   needforheat.Time(m.ServerTime),
		DeviceTime:   needforheat.Time(m.DeviceTime),
		Size:         m.Size,
		Measurements: measurements,
	}
}

func (r *UploadRepository) Find(upload upload.Upload) (upload.Upload, error) {
	uploadModel := MakeUploadModel(upload)
	err := r.db.Preload("Measurements").Where(&uploadModel).Find(&uploadModel).Error
	return uploadModel.fromModel(), err
}

func (r *UploadRepository) GetAll() ([]upload.Upload, error) {
	var uploads []upload.Upload

	var uploadModels []UploadModel
	err := r.db.Preload("Measurements").Find(&uploadModels).Error
	if err != nil {
		return nil, err
	}

	for _, uploadModel := range uploadModels {
		uploads = append(uploads, uploadModel.fromModel())
	}

	return uploads, nil
}

func (r *UploadRepository) Create(upload upload.Upload) (upload.Upload, error) {
	uploadModel := MakeUploadModel(upload)
	logrus.Info(uploadModel)
	err := r.db.Create(&uploadModel).Error
	logrus.Info(err)
	return uploadModel.fromModel(), err
}

func (r *UploadRepository) Delete(upload upload.Upload) error {
	uploadModel := MakeUploadModel(upload)
	return r.db.Delete(&uploadModel).Error
}

func (r *UploadRepository) GetLatestUploadForDeviceWithID(id uint) (upload.Upload, error) {
	var uploadModel UploadModel

	// Subquery to find upload IDs where the only measurements are those with the property name 'heartbeat'
	heartbeatOnlySubquery := r.db.
		Table("upload").
		Select("id").
		Where("instance_type = ? AND instance_id = ? AND size = (SELECT COUNT(*) FROM measurement WHERE upload_id = upload.id AND property_id = (SELECT id FROM property WHERE name = 'heartbeat'))", upload.Device, id)

	// Main query to fetch the latest upload model excluding those with only 'heartbeat' property measurements
	err := r.db.
		Where("instance_type = ? AND instance_id = ? AND id NOT IN (?)", upload.Device, id, heartbeatOnlySubquery).
		Order("server_time desc").
		First(&uploadModel).Error

	if err != nil {
		return upload.Upload{}, err
	}

	return uploadModel.fromModel(), nil
}

func StringToType(category string) upload.InstanceType {
	switch category {
	case "device":
		return upload.Device
	case "energy_query":
		return upload.EnergyQuery
	default:
		return ""
	}
}
