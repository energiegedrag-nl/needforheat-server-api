package services

import (
	"errors"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/apikey"
)

var (
	ErrAPIKeyTypeNameInvalid = errors.New("APIKey type name invalid")
)

type APIKeyService struct {
	repository apikey.APIKeyRepository
}

// Create a new APIKeyService.
func NewAPIKeyService(repository apikey.APIKeyRepository) *APIKeyService {
	return &APIKeyService{
		repository: repository,
	}
}

func (s *APIKeyService) Create(apiKey apikey.APIKey) (apikey.APIKey, error) {
	d := apikey.MakeAPIKey(apiKey.APIName, apiKey.APIKey)
	return s.repository.Create(d)
}

func (s *APIKeyService) Find(apiKey apikey.APIKey) (apikey.APIKey, error) {
	return s.repository.Find(apiKey)
}

func (s *APIKeyService) GetByID(id uint) (apikey.APIKey, error) {
	return s.repository.Find(apikey.APIKey{ID: id})
}

func (s *APIKeyService) Delete(apiKey apikey.APIKey) error {
	return s.repository.Delete(apiKey)
}
