package device

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
)

// A DeviceRepository can load, store and delete devices.
type DeviceRepository interface {
	Find(device Device) (Device, error)
	FindCloudFeedAuthCreationTimeFromDeviceID(deviceID uint) (*needforheat.Time, error)
	GetProperties(device Device) ([]property.Property, error)
	GetMeasurements(device Device, filters map[string]string) ([]measurement.Measurement, error)
	GetAll() ([]Device, error)
	Create(Device) (Device, error)
	Update(Device) (Device, error)
	Delete(Device) error
	GetAllByAccount(accountID uint) ([]Device, error)
}
