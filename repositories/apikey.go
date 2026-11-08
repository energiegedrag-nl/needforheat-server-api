package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/apikey"
	"gorm.io/gorm"
)

type APIKeyRepository struct {
	db *gorm.DB
}

// Create a new APIKeyRepository.
func NewAPIKeyRepository(db *gorm.DB) *APIKeyRepository {
	return &APIKeyRepository{
		db: db,
	}
}

// Database representation of a [apikey.APIKey]
type APIKeyModel struct {
	gorm.Model
	APIName string
	APIKey  string
}

// Set the name of the table in the database.
func (APIKeyModel) TableName() string {
	return "api_key"
}

// Create a APIKeyModel from a [apikey.APIKey].
func MakeAPIKeyModel(apiKey apikey.APIKey) APIKeyModel {
	return APIKeyModel{
		Model:   gorm.Model{ID: apiKey.ID},
		APIName: apiKey.APIName,
		APIKey:  apiKey.APIKey,
	}
}

// Create a [apikey.APIKey] from a APIKeyModel.
func (m *APIKeyModel) fromModel() apikey.APIKey {
	return apikey.APIKey{
		ID:      m.Model.ID,
		APIName: m.APIName,
		APIKey:  m.APIKey,
	}
}

func (r *APIKeyRepository) Find(apiKey apikey.APIKey) (apikey.APIKey, error) {
	apiKeyModel := MakeAPIKeyModel(apiKey)
	err := r.db.Where(&apiKeyModel).First(&apiKeyModel).Error
	return apiKeyModel.fromModel(), err
}

func (r *APIKeyRepository) GetAll() ([]apikey.APIKey, error) {
	var apiKeys []apikey.APIKey

	var apiKeyModels []APIKeyModel
	err := r.db.Find(&apiKeyModels).Error
	if err != nil {
		return nil, err
	}

	for _, apiKeyModel := range apiKeyModels {
		apiKeys = append(apiKeys, apiKeyModel.fromModel())
	}

	return apiKeys, nil
}

func (r *APIKeyRepository) Create(apiKey apikey.APIKey) (apikey.APIKey, error) {
	apiKeyModel := MakeAPIKeyModel(apiKey)
	err := r.db.Create(&apiKeyModel).Error
	return apiKeyModel.fromModel(), err
}

func (r *APIKeyRepository) Delete(apiKey apikey.APIKey) error {
	apiKeyModel := MakeAPIKeyModel(apiKey)
	return r.db.Delete(&apiKeyModel).Error
}
