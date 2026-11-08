package services

import (
	"errors"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquery"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/energyquerytype"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/measurement"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/upload"
)

var (
	ErrEnergyQueryDoesNotBelongToAccount = errors.New("EnergyQuery does not belong to this account")
	ErrEnergyQueryTypeNameInvalid        = errors.New("EnergyQuery type name invalid")
)

type EnergyQueryService struct {
	repository energyquery.EnergyQueryRepository

	// Services used when activating a EnergyQuery.
	authService *AuthorizationService

	// Services used when creating a EnergyQuery.
	energyQueryTypeService *EnergyQueryTypeService
	accountService         *AccountService

	// Services used when getting EnergyQuery info.
	uploadService *UploadService
}

// Create a new EnergyQueryService.
func NewEnergyQueryService(repository energyquery.EnergyQueryRepository, authService *AuthorizationService, energyQueryTypeService *EnergyQueryTypeService, AccountService *AccountService, uploadService *UploadService) *EnergyQueryService {
	return &EnergyQueryService{
		repository:             repository,
		authService:            authService,
		energyQueryTypeService: energyQueryTypeService,
		accountService:         AccountService,
		uploadService:          uploadService,
	}
}

func (s *EnergyQueryService) Create(queryType energyquerytype.EnergyQueryType, accountID uint, uploads []upload.Upload) (energyquery.EnergyQuery, error) {
	d := energyquery.MakeEnergyQuery(queryType, accountID, uploads)
	return s.repository.Create(d)
}

func (s *EnergyQueryService) GetByID(id uint) (energyquery.EnergyQuery, error) {
	return s.repository.Find(energyquery.EnergyQuery{ID: id})
}

func (s *EnergyQueryService) GetByTypeAndAccount(queryType energyquerytype.EnergyQueryType, accountID uint) (energyquery.EnergyQuery, error) {
	d, err := s.repository.Find(energyquery.EnergyQuery{EnergyQueryType: queryType, AccountID: accountID})
	if err != nil {
		return energyquery.EnergyQuery{}, err
	}

	return d, nil
}

func (s *EnergyQueryService) GetAccountByEnergyQueryID(id uint) (uint, error) {
	EnergyQuery, err := s.repository.Find(energyquery.EnergyQuery{ID: id})
	if err != nil {
		return 0, err
	}

	return EnergyQuery.AccountID, nil
}

func (s *EnergyQueryService) GetMeasurementsByEnergyQueryID(id uint, filters map[string]string) ([]measurement.Measurement, error) {
	measurements, err := s.repository.GetMeasurements(energyquery.EnergyQuery{ID: id}, filters)
	if err != nil {
		return nil, err
	}

	return measurements, nil
}

func (s *EnergyQueryService) GetPropertiesByEnergyQueryID(id uint) ([]property.Property, error) {
	properties, err := s.repository.GetProperties(energyquery.EnergyQuery{ID: id})
	if err != nil {
		return nil, err
	}

	return properties, nil
}

func (s *EnergyQueryService) GetAllByAccount(accountId uint) ([]energyquery.EnergyQuery, error) {
	energyQueries, err := s.repository.GetAllByAccount(accountId)
	if err != nil {
		return nil, err
	}

	return energyQueries, nil
}
