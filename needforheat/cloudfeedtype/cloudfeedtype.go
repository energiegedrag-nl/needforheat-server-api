package cloudfeedtype

import "github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"

// A CloudFeedType is an external online data source.
type CloudFeedType struct {
	ID               uint                  `json:"id"`
	Name             string                `json:"name"`
	AuthorizationURL string                `json:"authorization_url"`
	TokenURL         string                `json:"token_url"`
	ClientID         string                `json:"client_id"`
	ClientSecret     string                `json:"client_secret,omitempty"`
	Scope            string                `json:"scope"`
	RedirectURL      string                `json:"redirect_url"`
	CloudFeeds       []cloudfeed.CloudFeed `json:"-"`
}

// Create a new CloudFeedType.
func MakeCloudFeedType(name, authorizationURL, tokenURL, clientID, clientSecret, scope, redirectURL string) CloudFeedType {
	return CloudFeedType{
		Name:             name,
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		Scope:            scope,
		RedirectURL:      redirectURL,
	}
}
