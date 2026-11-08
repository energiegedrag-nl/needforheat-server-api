package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/internal/encryption"
	"github.com/energiegedrag-nl/needforheat-server-api/internal/helpers"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
	"gorm.io/gorm"
)

type CloudFeedRepository struct {
	db *gorm.DB
}

func NewCloudFeedRepository(db *gorm.DB) *CloudFeedRepository {
	return &CloudFeedRepository{
		db: db,
	}
}

// Database representation of [cloudfeed.CloudFeed].
type CloudFeedModel struct {
	AccountID       uint `gorm:"primaryKey;autoIncrement:false"`
	CloudFeedTypeID uint `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt       needforheat.Time
	UpdatedAt       needforheat.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
	// TODO: WARNING encrypted string encryption not yet implemented.
	AccessToken    encryption.EncryptedString
	RefreshToken   encryption.EncryptedString
	Expiry         needforheat.Time
	AuthGrantToken encryption.EncryptedString
	ActivatedAt    *needforheat.Time
}

// Set the name of the table in the database.
func (CloudFeedModel) TableName() string {
	return "cloud_feed"
}

// Create a new CloudFeedModel from a [needforheat.cloudFeed].
func MakeCloudFeedModel(cloudFeed cloudfeed.CloudFeed) CloudFeedModel {
	return CloudFeedModel{
		AccountID:       cloudFeed.AccountID,
		CloudFeedTypeID: cloudFeed.CloudFeedTypeID,
		AccessToken:     encryption.EncryptedString(cloudFeed.AccessToken),
		RefreshToken:    encryption.EncryptedString(cloudFeed.RefreshToken),
		Expiry:          cloudFeed.Expiry,
		AuthGrantToken:  encryption.EncryptedString(cloudFeed.AuthGrantToken),
		ActivatedAt:     cloudFeed.ActivatedAt,
	}
}

// Create a [cloudfeed.CloudFeed] from an CloudFeedModel.
func (m *CloudFeedModel) fromModel() cloudfeed.CloudFeed {
	return cloudfeed.CloudFeed{
		AccountID:       m.AccountID,
		CloudFeedTypeID: m.CloudFeedTypeID,
		AccessToken:     string(m.AccessToken),
		RefreshToken:    string(m.RefreshToken),
		Expiry:          m.Expiry,
		AuthGrantToken:  string(m.AuthGrantToken),
		ActivatedAt:     m.ActivatedAt,
	}
}

func (r *CloudFeedRepository) Find(cloudFeed cloudfeed.CloudFeed) (cloudfeed.CloudFeed, error) {
	cloudFeedModel := MakeCloudFeedModel(cloudFeed)
	err := r.db.Where(&cloudFeedModel).First(&cloudFeedModel).Error
	return cloudFeedModel.fromModel(), err
}

func (r *CloudFeedRepository) FindOAuthInfo(accountID uint, cloudFeedID uint) (string, string, string, string, error) {
	var result struct {
		TokenURL     string
		RefreshToken string
		ClientID     string
		ClientSecret string
	}
	err := r.db.Table("cloud_feed_type").Select("cloud_feed_type.token_url, cloud_feed.refresh_token AS refresh_token, cloud_feed_type.client_id, cloud_feed_type.client_secret").Joins("JOIN cloud_feed ON cloud_feed_type.id = cloud_feed.cloud_feed_type_id").Where("cloud_feed.account_id = ? AND cloud_feed.cloud_feed_type_id = ?", accountID, cloudFeedID).Scan(&result).Error
	return result.TokenURL, result.RefreshToken, result.ClientID, result.ClientSecret, err
}

func (r *CloudFeedRepository) FindFirstTokenToExpire() (uint, uint, needforheat.Time, error) {
	var cloudFeedModel CloudFeedModel
	err := r.db.Order("expiry ASC").Where("expiry <> ''").First(&cloudFeedModel).Error
	return cloudFeedModel.AccountID, cloudFeedModel.CloudFeedTypeID, cloudFeedModel.Expiry, err
}

func (r *CloudFeedRepository) FindDevice(cloudFeed cloudfeed.CloudFeed) (*device.Device, error) {
	var device DeviceModel

	err := r.db.Model(&device).
		Joins("JOIN device_type ON device_type.id = device.device_type_id").
		Joins("JOIN cloud_feed_type ON cloud_feed_type.name = device_type.name").
		Joins("JOIN cloud_feed ON cloud_feed.cloud_feed_type_id = cloud_feed_type.id").
		Joins("JOIN account ON account.id = device.account_id").
		Where("account.id = cloud_feed.account_id AND cloud_feed.account_id = ? AND cloud_feed.cloud_feed_type_id = ?", cloudFeed.AccountID, cloudFeed.CloudFeedTypeID).
		First(&device).
		Error

	if err != nil {
		return nil, err
	}

	d := device.fromModel()
	return &d, nil
}

func (r *CloudFeedRepository) GetAll() ([]cloudfeed.CloudFeed, error) {
	var cloudFeeds []cloudfeed.CloudFeed

	var cloudFeedModels []CloudFeedModel
	err := r.db.Find(&cloudFeedModels).Error
	if err != nil {
		return nil, err
	}

	for _, cloudFeedModel := range cloudFeedModels {
		cloudFeeds = append(cloudFeeds, cloudFeedModel.fromModel())
	}

	return cloudFeeds, nil
}

func (r *CloudFeedRepository) Create(cloudFeed cloudfeed.CloudFeed) (cloudfeed.CloudFeed, error) {
	// First check if a soft deleted entry exists.
	// We need to do this because we can't create a new one if one exists.
	cfaCheck := CloudFeedModel{}
	err := r.db.Unscoped().Where(&CloudFeedModel{
		AccountID:       cloudFeed.AccountID,
		CloudFeedTypeID: cloudFeed.CloudFeedTypeID,
	}).First(&cfaCheck).Error

	// Return immediately if there is an error,
	// except if the error was RecordNotFound.
	if err != nil && !helpers.IsMySQLRecordNotFoundError(err) {
		return cloudFeed, err
	}

	// At this point we know that there are no errors,
	// except maybe that the record was not found, so check.
	if !helpers.IsMySQLRecordNotFoundError(err) {
		// Record was found. Check if it was soft deleted.
		if cfaCheck.DeletedAt.Valid {
			// Record was soft deleted. Delete it so we can create a new one.
			err := r.db.Unscoped().Delete(&cfaCheck).Error
			if err != nil {
				return cloudFeed, err
			}
		}
	}

	// If we reach this, it means there was not previous record,
	// or it was deleted, and we can just create a new one.
	cloudFeedModel := MakeCloudFeedModel(cloudFeed)
	err = r.db.Create(&cloudFeedModel).Error
	return cloudFeedModel.fromModel(), err
}

func (r *CloudFeedRepository) Update(cloudFeed cloudfeed.CloudFeed) (cloudfeed.CloudFeed, error) {
	cloudFeedModel := MakeCloudFeedModel(cloudFeed)
	err := r.db.Model(&cloudFeedModel).Updates(cloudFeedModel).Error
	return cloudFeedModel.fromModel(), err
}

func (r *CloudFeedRepository) Delete(cloudFeed cloudfeed.CloudFeed) error {
	CloudFeedAuthModel := MakeCloudFeedModel(cloudFeed)
	return r.db.Delete(&CloudFeedAuthModel).Error
}
