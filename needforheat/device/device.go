package device

import (
	"errors"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/devicetype"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrDeviceActivationSecretIncorrect = errors.New("device activation_secret is incorrect")
)

// A Device is collects measurements in a subject's account.
type Device struct {
	ID                   uint                  `json:"id"`
	Name                 string                `json:"name"`
	DeviceType           devicetype.DeviceType `json:"device_type"`
	AccountID            uint                  `json:"account_id"`
	ActivationSecret     string                `json:"activation_secret,omitempty"` // This can be removed if a device uses JWT's too.
	ActivationSecretHash string                `json:"-"`                           // This can be removed if a device uses JWT's too.
	ActivatedAt          *needforheat.Time     `json:"activated_at"`
	AuthorizationToken   string                `json:"authorization_token,omitempty"`
	Uploads              []upload.Upload       `json:"uploads,omitempty"`
	LatestUpload         *needforheat.Time     `json:"latest_upload,omitempty"`
}

// Create a new Device.
func MakeDevice(name string, deviceType devicetype.DeviceType, accountID uint, activationSecret string) Device {
	activationSecretHash, err := bcrypt.GenerateFromPassword([]byte(activationSecret), 12)
	if err != nil {
		logrus.Error("a device was created, but activationSecretHash could not be generated")
	}

	now := time.Now().Unix()
	activatedAt := needforheat.Time(time.Unix(now, 0))

	// Hack, so monitoring works when you add a new ENelogic
	if deviceType.Name == "enelogic" {
		return Device{
			Name:                 name,
			DeviceType:           deviceType,
			AccountID:            accountID,
			ActivationSecretHash: string(activationSecretHash),
			ActivatedAt:          &activatedAt,
		}
	} else {
		return Device{
			Name:                 name,
			DeviceType:           deviceType,
			AccountID:            accountID,
			ActivationSecretHash: string(activationSecretHash),
		}
	}
}

// Activate a device.
func (d *Device) Activate(activationSecret string) error {
	if activationSecret == "" || bcrypt.CompareHashAndPassword([]byte(d.ActivationSecretHash), []byte(activationSecret)) != nil {
		return ErrDeviceActivationSecretIncorrect
	}

	now := time.Now().Unix()
	activatedAt := needforheat.Time(time.Unix(now, 0))
	d.ActivatedAt = &activatedAt

	return nil
}

// Add an upload to a device.
func (d *Device) AddUpload(upload upload.Upload) {
	d.Uploads = append(d.Uploads, upload)
}
