package cloudfeed

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
)

// A CloudFeedRepository can load, store and delete CloudFeeds.
type CloudFeedRepository interface {
	Find(CloudFeed CloudFeed) (CloudFeed, error)
	FindOAuthInfo(accountID uint, cloudFeedID uint) (tokenURL string, refreshToken string, clientID string, clientSecret string, err error)
	FindFirstTokenToExpire() (accountID uint, cloudFeedID uint, expiry needforheat.Time, err error)
	FindDevice(CloudFeed CloudFeed) (*device.Device, error)
	GetAll() ([]CloudFeed, error)
	Create(CloudFeed) (CloudFeed, error)
	Update(CloudFeed) (CloudFeed, error)
	Delete(CloudFeed) error
}
