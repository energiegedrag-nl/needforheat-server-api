package account

import (
	"errors"
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/cloudfeed"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/device"
)

var (
	ErrAccountAlreadyActivated = errors.New("account is already activated")
)

// An Account is registered to a research subject.
type Account struct {
	ID                 uint                  `json:"id"`
	Campaign           campaign.Campaign     `json:"campaign"`
	ActivatedAt        *needforheat.Time     `json:"activated_at"`
	InvitationToken    string                `json:"invitation_token,omitempty"`
	InvitationURL      string                `json:"invitation_url,omitempty"`
	AuthorizationToken string                `json:"authorization_token,omitempty"`
	Devices            []*device.Device      `json:"devices,omitempty"`
	CloudFeeds         []cloudfeed.CloudFeed `json:"cloud_feeds,omitempty"`
	// Maybe use separate pseudonym field,
	// but right now we can derive a pseudonym
	// using the ID or the campaign ID + account ID.
}

// Create a new Account.
func MakeAccount(campaign campaign.Campaign) Account {
	return Account{
		Campaign: campaign,
	}
}

// Activate an account.
// An error will be returned if the account is already activated.
func (a *Account) Activate() error {
	if a.ActivatedAt != nil {
		return ErrAccountAlreadyActivated
	}

	now := time.Now().Unix()
	activatedAt := needforheat.Time(time.Unix(now, 0))
	a.ActivatedAt = &activatedAt

	return nil
}
