package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/internal/encryption"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeedtype"
	"gorm.io/gorm"
)

type CloudFeedTypeRepository struct {
	db *gorm.DB
}

// Create a new CloudFeedTypeRepository.
func NewCloudFeedTypeRepository(db *gorm.DB) *CloudFeedTypeRepository {
	return &CloudFeedTypeRepository{
		db: db,
	}
}

// Database representation of a [cloudfeed.CloudFeed]
type CloudFeedTypeModel struct {
	gorm.Model
	Name             string `gorm:"unique;not null"`
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	// TODO: WARNING EncryptedString still has to implement the encryption.
	ClientSecret    encryption.EncryptedString
	Scope           string
	RedirectURL     string
	CloudFeeds      []CloudFeedModel      `gorm:"foreignKey:CloudFeedTypeID"`
	DataSourceTypes []DataSourceTypeModel `gorm:"polymorphic:TypeInstance;"`
}

// Set the name of the table in the database.
func (CloudFeedTypeModel) TableName() string {
	return "cloud_feed_type"
}

// Create a CloudFeedTypeModel from a [cloudfeedtype.CloudFeedtype].
func MakeCloudFeedTypeModel(cloudFeedType cloudfeedtype.CloudFeedType) CloudFeedTypeModel {
	return CloudFeedTypeModel{
		Model:            gorm.Model{ID: cloudFeedType.ID},
		Name:             cloudFeedType.Name,
		AuthorizationURL: cloudFeedType.AuthorizationURL,
		TokenURL:         cloudFeedType.TokenURL,
		ClientID:         cloudFeedType.ClientID,
		ClientSecret:     encryption.EncryptedString(cloudFeedType.ClientSecret),
		Scope:            cloudFeedType.Scope,
		RedirectURL:      cloudFeedType.RedirectURL,
	}
}

// Create a [cloudfeedtype.CloudFeedType] from a CloudFeedTypeModel.
func (m *CloudFeedTypeModel) fromModel() cloudfeedtype.CloudFeedType {
	return cloudfeedtype.CloudFeedType{
		ID:               m.Model.ID,
		Name:             m.Name,
		AuthorizationURL: m.AuthorizationURL,
		TokenURL:         m.TokenURL,
		ClientID:         m.ClientID,
		ClientSecret:     string(m.ClientSecret),
		Scope:            m.Scope,
		RedirectURL:      m.RedirectURL,
	}
}

func (r *CloudFeedTypeRepository) Find(cloudFeedType cloudfeedtype.CloudFeedType) (cloudfeedtype.CloudFeedType, error) {
	cloudFeedTypeModel := MakeCloudFeedTypeModel(cloudFeedType)
	err := r.db.Where(&cloudFeedTypeModel).First(&cloudFeedTypeModel).Error
	return cloudFeedTypeModel.fromModel(), err
}

func (r *CloudFeedTypeRepository) GetAll() ([]cloudfeedtype.CloudFeedType, error) {
	var cloudFeedTypes []cloudfeedtype.CloudFeedType

	var cloudFeedTypeModels []CloudFeedTypeModel
	err := r.db.Find(&cloudFeedTypeModels).Error
	if err != nil {
		return nil, err
	}

	for _, cloudFeedTypeModel := range cloudFeedTypeModels {
		cloudFeedTypes = append(cloudFeedTypes, cloudFeedTypeModel.fromModel())
	}

	return cloudFeedTypes, nil
}

func (r *CloudFeedTypeRepository) Create(cloudFeedType cloudfeedtype.CloudFeedType) (cloudfeedtype.CloudFeedType, error) {
	cloudFeedTypeModel := MakeCloudFeedTypeModel(cloudFeedType)
	err := r.db.Create(&cloudFeedTypeModel).Error
	return cloudFeedTypeModel.fromModel(), err
}

func (r *CloudFeedTypeRepository) Update(cloudFeedType cloudfeedtype.CloudFeedType) (cloudfeedtype.CloudFeedType, error) {
	cloudFeedTypeModel := MakeCloudFeedTypeModel(cloudFeedType)
	err := r.db.Model(&cloudFeedTypeModel).Updates(cloudFeedTypeModel).Error
	return cloudFeedTypeModel.fromModel(), err
}

func (r *CloudFeedTypeRepository) Delete(cloudFeedType cloudfeedtype.CloudFeedType) error {
	cloudFeedTypeModel := MakeCloudFeedTypeModel(cloudFeedType)
	return r.db.Delete(&cloudFeedTypeModel).Error
}
