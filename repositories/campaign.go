package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
	"gorm.io/gorm"
)

type CampaignRepository struct {
	db *gorm.DB
}

func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{
		db: db,
	}
}

// Database representation of [campaign.Campaign].
type CampaignModel struct {
	gorm.Model
	Name             string `gorm:"unique;not null"`
	AppModelID       uint   `gorm:"column:app_id"`
	App              AppModel
	InfoURL          string `gorm:"unique;not null"`
	StartTime        *needforheat.Time
	EndTime          *needforheat.Time
	DataSourceListID uint
}

// Set the name of the table in the database.
func (CampaignModel) TableName() string {
	return "campaign"
}

// Create a new CampaignModel from a [needforheat.campaign].
func MakeCampaignModel(campaign campaign.Campaign) CampaignModel {
	return CampaignModel{
		Model: gorm.Model{
			ID: campaign.ID,
		},
		Name:             campaign.Name,
		AppModelID:       campaign.App.ID,
		App:              MakeAppModel(campaign.App),
		InfoURL:          campaign.InfoURL,
		StartTime:        campaign.StartTime,
		EndTime:          campaign.EndTime,
		DataSourceListID: campaign.DataSourceList.ID,
	}
}

// Create a [campaign.Campaign] from an CampaignModel.
func (m *CampaignModel) fromModel() campaign.Campaign {
	return campaign.Campaign{
		ID:        m.ID,
		Name:      m.Name,
		App:       m.App.fromModel(),
		InfoURL:   m.InfoURL,
		StartTime: m.StartTime,
		EndTime:   m.EndTime,
	}
}

func (r *CampaignRepository) Find(campaignToFind campaign.Campaign) (campaign.Campaign, error) {
	campaignModel := MakeCampaignModel(campaignToFind)
	err := r.db.Preload("App").Where(&campaignModel).First(&campaignModel).Error

	var dataSourceList DataSourceListModel
	dsErr := r.db.Preload("Items").Where("id = ?", campaignModel.DataSourceListID).First(&dataSourceList).Error
	if dsErr != nil {
		return campaign.Campaign{}, dsErr
	}

	campaignAPI := campaignModel.fromModel()
	campaignAPI.DataSourceList = dataSourceList.fromModel(r.db)

	return campaignModel.fromModel(), err
}

func (r *CampaignRepository) GetAll() ([]campaign.Campaign, error) {
	var campaigns []campaign.Campaign

	var campaignModels []CampaignModel
	err := r.db.Preload("App").Find(&campaignModels).Error
	if err != nil {
		return nil, err
	}

	for _, campaignModel := range campaignModels {
		var dataSourceList DataSourceListModel
		dsErr := r.db.Preload("Items").Where("id = ?", campaignModel.DataSourceListID).First(&dataSourceList).Error
		if dsErr != nil {
			return nil, dsErr
		}
		campaignAPI := campaignModel.fromModel()
		campaignAPI.DataSourceList = dataSourceList.fromModel(r.db)
		campaigns = append(campaigns, campaignAPI)
	}

	return campaigns, nil
}

func (r *CampaignRepository) Create(campaign campaign.Campaign) (campaign.Campaign, error) {
	campaignModel := MakeCampaignModel(campaign)
	err := r.db.Create(&campaignModel).Error
	return campaignModel.fromModel(), err
}

func (r *CampaignRepository) Delete(campaign campaign.Campaign) error {
	CampaignModel := MakeCampaignModel(campaign)
	return r.db.Delete(&CampaignModel).Error
}
