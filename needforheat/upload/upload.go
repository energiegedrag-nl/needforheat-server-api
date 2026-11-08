package upload

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
)

// An Upload is a collection of measurements, with additional information.
type Upload struct {
	ID           uint                      `json:"id"`
	InstanceID   uint                      `json:"instance_id"`
	InstanceType InstanceType              `json:"instance_type"`
	ServerTime   needforheat.Time          `json:"server_time"`
	DeviceTime   needforheat.Time          `json:"device_time"`
	Size         int                       `json:"size"`
	Measurements []measurement.Measurement `json:"measurements,omitempty"`
}

type InstanceType string

const (
	Device      InstanceType = "device"
	EnergyQuery InstanceType = "energy_query"
)

// Create a new Upload.
func MakeUpload(instanceID uint, instanceType InstanceType, deviceTime needforheat.Time, measurements []measurement.Measurement) Upload {
	return Upload{
		InstanceID:   instanceID,
		InstanceType: instanceType,
		ServerTime:   needforheat.Time(time.Now().UTC()),
		DeviceTime:   deviceTime,
		Size:         len(measurements),
		Measurements: measurements,
	}
}
