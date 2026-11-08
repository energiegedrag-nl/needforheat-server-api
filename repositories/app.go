package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/app"
	"gorm.io/gorm"
)

type AppRepository struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) *AppRepository {
	return &AppRepository{
		db: db,
	}
}

// Database representation of a [app.App].
type AppModel struct {
	gorm.Model
	Name                    string `gorm:"unique;not null"`
	ProvisioningURLTemplate string
	OauthRedirectURL        string
}

// Set the name of the table in the database.
func (AppModel) TableName() string {
	return "app"
}

// Create a new AppModel from a [app.App]
func MakeAppModel(app app.App) AppModel {
	return AppModel{
		Model:                   gorm.Model{ID: app.ID},
		Name:                    app.Name,
		ProvisioningURLTemplate: app.ProvisioningURLTemplate,
		OauthRedirectURL:        app.OauthRedirectURL,
	}
}

// Create a [app.App] from an AppModel.
func (m *AppModel) fromModel() app.App {
	return app.App{
		ID:                      m.Model.ID,
		Name:                    m.Name,
		ProvisioningURLTemplate: m.ProvisioningURLTemplate,
		OauthRedirectURL:        m.OauthRedirectURL,
	}
}

func (r *AppRepository) Find(app app.App) (app.App, error) {
	appModel := MakeAppModel(app)
	err := r.db.Where(&appModel).First(&appModel).Error
	return appModel.fromModel(), err
}

func (r *AppRepository) GetAll() ([]app.App, error) {
	apps := make([]app.App, 0)

	var appModels []AppModel
	err := r.db.Find(&appModels).Error
	if err != nil {
		return nil, err
	}

	for _, appModel := range appModels {
		apps = append(apps, appModel.fromModel())
	}

	return apps, nil
}

func (r *AppRepository) Create(app app.App) (app.App, error) {
	appModel := MakeAppModel(app)
	err := r.db.Create(&appModel).Error
	return appModel.fromModel(), err
}

func (r *AppRepository) Delete(app app.App) error {
	appModel := MakeAppModel(app)
	return r.db.Delete(&appModel).Error
}
