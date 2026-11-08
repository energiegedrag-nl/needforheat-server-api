package services

import (
	"errors"
	"fmt"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"github.com/sigurn/crc16"
	"github.com/sirupsen/logrus"
)

var (
	ErrHashDoesNotMatchEnergyQueryType = errors.New("hash does not match a energy query type")
)

type EnergyQueryTypeService struct {
	repository energyquerytype.EnergyQueryTypeRepository

	// Service used when creating a device type.
	propertyService *PropertyService

	// Hashed device types.
	hashedEnergyQueryTypes map[string]string
}

// Create a new EnergyQueryTypeService.
func NewEnergyQueryTypeService(repository energyquerytype.EnergyQueryTypeRepository, propertyService *PropertyService) *EnergyQueryTypeService {
	EnergyQueryTypeService := &EnergyQueryTypeService{
		repository:      repository,
		propertyService: propertyService,
	}

	EnergyQueryTypeService.updateEnergyQueryTypeHashes()

	return EnergyQueryTypeService
}

func (s *EnergyQueryTypeService) Create(variety string, formula string) (energyquerytype.EnergyQueryType, error) {
	EnergyQueryType := energyquerytype.MakeEnergyQueryType(variety, formula)

	EnergyQueryType, err := s.repository.Create(EnergyQueryType)
	if err != nil {
		return EnergyQueryType, err
	}

	s.updateEnergyQueryTypeHashes()

	return EnergyQueryType, nil
}

func (s *EnergyQueryTypeService) Find(energyQueryType energyquerytype.EnergyQueryType) (energyquerytype.EnergyQueryType, error) {
	return s.repository.Find(energyQueryType)
}

func (s *EnergyQueryTypeService) GetByHash(energyQueryTypeHash string) (energyquerytype.EnergyQueryType, error) {
	variety, ok := s.hashedEnergyQueryTypes[energyQueryTypeHash]
	if !ok {
		return energyquerytype.EnergyQueryType{}, ErrHashDoesNotMatchEnergyQueryType
	}

	return s.repository.Find(energyquerytype.EnergyQueryType{EnergyQueryVariety: variety})
}

func (s *EnergyQueryTypeService) GetByID(id uint) (energyquerytype.EnergyQueryType, error) {
	return s.repository.Find(energyquerytype.EnergyQueryType{ID: id})
}

func (s *EnergyQueryTypeService) GetByIDForDataSourceType(id uint) (interface{}, error) {
	return s.repository.Find(energyquerytype.EnergyQueryType{ID: id})
}

func (s *EnergyQueryTypeService) GetByVariety(variety string) (energyquerytype.EnergyQueryType, error) {
	return s.repository.Find(energyquerytype.EnergyQueryType{EnergyQueryVariety: variety})
}

func (s *EnergyQueryTypeService) GetTableName() string {
	return "energy_query_type"
}

// Update the map of hashes to device types.
func (s *EnergyQueryTypeService) updateEnergyQueryTypeHashes() {
	EnergyQueryTypes, err := s.repository.GetAll()
	if err != nil {
		logrus.Warn(err)
		return
	}

	s.hashedEnergyQueryTypes = make(map[string]string)

	table := crc16.MakeTable(crc16.CRC16_XMODEM)

	for _, EnergyQueryType := range EnergyQueryTypes {
		hash := crc16.Checksum([]byte(EnergyQueryType.EnergyQueryVariety), table)
		s.hashedEnergyQueryTypes[fmt.Sprintf("%X", hash)] = EnergyQueryType.EnergyQueryVariety
	}
}
