package services

import (
	"errors"

	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
)

var (
	ErrEmptyUpload = errors.New("no measurements in upload")
)

type UploadService struct {
	repository upload.UploadRepository
	deviceRepo device.DeviceRepository

	// Service used when creating an upload.
	propertyService *PropertyService
}

// Create a new UploadService.
func NewUploadService(
	repository upload.UploadRepository,
	deviceRepo device.DeviceRepository,
	propertyService *PropertyService,
) *UploadService {
	return &UploadService{
		repository:      repository,
		deviceRepo:      deviceRepo,
		propertyService: propertyService,
	}
}

func (s *UploadService) Create(instanceID uint, instanceType upload.InstanceType, deviceTime needforheat.Time, measurements []measurement.Measurement) (upload.Upload, error) {
	if len(measurements) <= 0 {
		return upload.Upload{}, ErrEmptyUpload
	}

	//For older firmwares
	if instanceType == "" {
		instanceType = upload.Device
	}

	upload := upload.MakeUpload(instanceID, instanceType, deviceTime, measurements)

	upload, err := s.repository.Create(upload)

	return upload, err
}

func (s *UploadService) GetLatestUploadTimeForDeviceWithID(id uint) (*needforheat.Time, bool, error) {
	upload, err := s.repository.GetLatestUploadForDeviceWithID(id)

	if err != nil {
		// If the record is not found, there was no upload. That's not an error.
		if helpers.IsMySQLRecordNotFoundError(err) {
			uploadTime, err := s.getCloudFeedAuthCreationTimeForDeviceWithID(id)
			return uploadTime, false, err
		}
		return nil, false, err
	}

	return (*needforheat.Time)(&upload.ServerTime), true, nil
}

func (s *UploadService) getCloudFeedAuthCreationTimeForDeviceWithID(id uint) (*needforheat.Time, error) {
	creationTime, err := s.deviceRepo.FindCloudFeedAuthCreationTimeFromDeviceID(id)
	if err != nil && !helpers.IsMySQLRecordNotFoundError(err) {
		return nil, err
	}
	return creationTime, nil
}
