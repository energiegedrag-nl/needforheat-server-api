package repositories

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"gorm.io/gorm"
)

// Database representation of a [measurement.Measurement]
type MeasurementModel struct {
	gorm.Model
	PropertyModelID uint `gorm:"column:property_id;index:idx_measurement_upload_property,priority:2"`
	Property        PropertyModel
	UploadModelID   uint `gorm:"column:upload_id;index:idx_measurement_upload_property,priority:1"`
	Time            time.Time
	Value           string
}

// Set the name of the table in the database.
func (MeasurementModel) TableName() string {
	return "measurement"
}

// Create a MeasurementModel from a [measurement.Measurement].
func MakeMeasurementModel(measurement measurement.Measurement) MeasurementModel {
	return MeasurementModel{
		Model:           gorm.Model{ID: measurement.ID},
		PropertyModelID: measurement.Property.ID,
		Property:        MakePropertyModel(measurement.Property),
		UploadModelID:   measurement.UploadID,
		Time:            time.Time(measurement.Time),
		Value:           measurement.Value,
	}
}

// Create a [measurement.Measurement] from a MeasurementModel.
func (m *MeasurementModel) fromModel() measurement.Measurement {
	return measurement.Measurement{
		ID:       m.Model.ID,
		UploadID: m.UploadModelID,
		Property: m.Property.fromModel(),
		Time:     needforheat.Time(m.Time),
		Value:    m.Value,
	}
}
