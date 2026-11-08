package services

import (
	"errors"
	"fmt"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/devicetype"
	"github.com/sigurn/crc16"
	"github.com/sirupsen/logrus"
)

var (
	ErrHashDoesNotMatchType = errors.New("hash does not match a device type")
)

type DeviceTypeService struct {
	repository devicetype.DeviceTypeRepository

	// Service used when creating a device type.
	propertyService *PropertyService

	// Hashed device types.
	hashedDeviceTypes map[string]string
}

// Create a new DeviceTypeService.
func NewDeviceTypeService(repository devicetype.DeviceTypeRepository, propertyService *PropertyService) *DeviceTypeService {
	deviceTypeService := &DeviceTypeService{
		repository:      repository,
		propertyService: propertyService,
	}

	deviceTypeService.updateDeviceTypeHashes()

	return deviceTypeService
}

func (s *DeviceTypeService) Create(name string) (devicetype.DeviceType, error) {
	deviceType := devicetype.MakeDeviceType(name)

	deviceType, err := s.repository.Create(deviceType)
	if err != nil {
		return deviceType, err
	}

	s.updateDeviceTypeHashes()

	return deviceType, nil
}

func (s *DeviceTypeService) Find(deviceType devicetype.DeviceType) (devicetype.DeviceType, error) {
	return s.repository.Find(deviceType)
}

func (s *DeviceTypeService) GetByHash(deviceTypeHash string) (devicetype.DeviceType, error) {
	name, ok := s.hashedDeviceTypes[deviceTypeHash]
	if !ok {
		return devicetype.DeviceType{}, ErrHashDoesNotMatchType
	}

	return s.repository.Find(devicetype.DeviceType{Name: name})
}

func (s *DeviceTypeService) GetByID(id uint) (devicetype.DeviceType, error) {
	return s.repository.Find(devicetype.DeviceType{ID: id})
}

func (s *DeviceTypeService) GetByIDForDataSourceType(id uint) (interface{}, error) {
	return s.repository.Find(devicetype.DeviceType{ID: id})
}

func (s *DeviceTypeService) GetByName(name string) (devicetype.DeviceType, error) {
	return s.repository.Find(devicetype.DeviceType{Name: name})
}

func (s *DeviceTypeService) GetTableName() string {
	return "device_type"
}

// Update the map of hashes to device types.
func (s *DeviceTypeService) updateDeviceTypeHashes() {
	deviceTypes, err := s.repository.GetAll()
	if err != nil {
		logrus.Warn(err)
		return
	}

	s.hashedDeviceTypes = make(map[string]string)

	table := crc16.MakeTable(crc16.CRC16_XMODEM)

	for _, deviceType := range deviceTypes {
		hash := crc16.Checksum([]byte(deviceType.Name), table)
		s.hashedDeviceTypes[fmt.Sprintf("%X", hash)] = deviceType.Name
	}
}
