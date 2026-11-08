package services

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/app"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcelist"
	"github.com/sirupsen/logrus"
)

type CampaignService struct {
	repository campaign.CampaignRepository

	// Service used when creating a campaign.
	appService            *AppService
	dataSourceListService *DataSourceListService
}

// Create a new CampaignService.
func NewCampaignService(
	repository campaign.CampaignRepository,
	appService *AppService,
	dataSourceListService *DataSourceListService,
) *CampaignService {
	return &CampaignService{
		repository:            repository,
		appService:            appService,
		dataSourceListService: dataSourceListService,
	}
}

// Create a new campaign.
func (s *CampaignService) Create(
	name string,
	app app.App,
	infoURL string,
	startTime,
	endTime *needforheat.Time,
	dataSourceList datasourcelist.DataSourceList,
) (campaign.Campaign, error) {
	app, err := s.appService.Find(app)
	if err != nil {
		return campaign.Campaign{}, err
	}

	foundDataSourceList, err := s.dataSourceListService.Find(dataSourceList)
	if err != nil {
		return campaign.Campaign{}, err
	}
	logrus.Info(foundDataSourceList)
	campaign := campaign.MakeCampaign(name, app, infoURL, startTime, endTime, foundDataSourceList)

	campaignCreated, err := s.repository.Create(campaign)
	campaignCreated.DataSourceList = foundDataSourceList

	return campaignCreated, err
}

// Find a campaign using any field set in the campaign struct.
func (s *CampaignService) Find(campaign campaign.Campaign) (campaign.Campaign, error) {
	return s.repository.Find(campaign)
}

// Get a campaign by its ID.
func (s *CampaignService) GetByID(id uint) (campaign.Campaign, error) {
	return s.repository.Find(campaign.Campaign{ID: id})
}
