package services

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/app"
)

type AppService struct {
	repository app.AppRepository
}

// Create a new AppService.
func NewAppService(repository app.AppRepository) *AppService {
	return &AppService{
		repository: repository,
	}
}

// Create a new app.
func (s *AppService) Create(name, provisioningURLTemplate, oauthRedirectURL string) (app.App, error) {
	app := app.MakeApp(name, provisioningURLTemplate, oauthRedirectURL)
	return s.repository.Create(app)
}

func (s *AppService) Find(app app.App) (app.App, error) {
	return s.repository.Find(app)
}

func (s *AppService) GetAll() ([]app.App, error) {
	return s.repository.GetAll()
}

// Get an app by its ID.
func (s *AppService) GetByID(id uint) (app.App, error) {
	return s.repository.Find(app.App{ID: id})
}
