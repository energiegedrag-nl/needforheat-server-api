package cloudfeedstatus

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeedtype"
)

// A CloudFeedStatus contains all available cloud feeds for an account and if they are connected or not.
type CloudFeedStatus struct {
	CloudFeedType cloudfeedtype.CloudFeedType `json:"cloud_feed_type"`
	Connected     bool                        `json:"connected"`
}

// Create a new CloudFeedStatus.
func MakeCloudFeedStatus(cloudFeedType cloudfeedtype.CloudFeedType, cloudFeed cloudfeed.CloudFeed) CloudFeedStatus {
	// TODO: Change this to check if access and/or refresh token is valid.
	// For now, we use the grant token, since we have not implemented functionality
	// to exchange the  grant token for an access and refresh token.
	connected := cloudFeed.AuthGrantToken != ""

	return CloudFeedStatus{
		CloudFeedType: cloudFeedType,
		Connected:     connected,
	}
}
