package cloudfeed

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
)

// A CloudFeed stores auth information about CloudFeeds authorized by an account.
type CloudFeed struct {
	AccountID       uint              `json:"account_id"`
	CloudFeedTypeID uint              `json:"cloud_feed_id"`
	AccessToken     string            `json:"-"`
	RefreshToken    string            `json:"-"`
	Expiry          needforheat.Time  `json:"-"`
	AuthGrantToken  string            `json:"auth_grant_token"`
	ActivatedAt     *needforheat.Time `json:"activated_at"`
}

// Create a new CloudFeed.
func MakeCloudFeed(accountID, cloudFeedTypeID uint, accessToken string, refreshToken string, expiry needforheat.Time, authGrantToken string) CloudFeed {
	now := time.Now().Unix()
	activatedAt := needforheat.Time(time.Unix(now, 0))
	return CloudFeed{
		AccountID:       accountID,
		CloudFeedTypeID: cloudFeedTypeID,
		AccessToken:     accessToken,
		RefreshToken:    refreshToken,
		Expiry:          expiry,
		AuthGrantToken:  authGrantToken,
		ActivatedAt:     &activatedAt,
	}
}
