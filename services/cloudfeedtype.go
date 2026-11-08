package services

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeedtype"
)

type CloudFeedTypeService struct {
	repository cloudfeedtype.CloudFeedTypeRepository
}

// Create a new CloudFeedTypeService.
func NewCloudFeedTypeService(repository cloudfeedtype.CloudFeedTypeRepository) *CloudFeedTypeService {
	return &CloudFeedTypeService{
		repository: repository,
	}
}

func (s *CloudFeedTypeService) Create(name string, authorizationURL string, tokenURL string, clientID string, clientSecret string, scope string, redirectURL string) (cloudfeedtype.CloudFeedType, error) {
	cloudFeed := cloudfeedtype.MakeCloudFeedType(name, authorizationURL, tokenURL, clientID, clientSecret, scope, redirectURL)
	return s.repository.Create(cloudFeed)
}

func (s *CloudFeedTypeService) Find(cloudFeed cloudfeedtype.CloudFeedType) (cloudfeedtype.CloudFeedType, error) {
	return s.repository.Find(cloudFeed)
}

func (s *CloudFeedTypeService) GetByID(id uint) (cloudfeedtype.CloudFeedType, error) {
	return s.repository.Find(cloudfeedtype.CloudFeedType{ID: id})
}

func (s *CloudFeedTypeService) GetByIDForDataSourceType(id uint) (interface{}, error) {
	return s.repository.Find(cloudfeedtype.CloudFeedType{ID: id})
}

func (s *CloudFeedTypeService) GetTableName() string {
	return "cloud_feed_type"
}
