package propertylabel

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/campaign"
)

// A PropertyLabel is the display name of a property (such as "heat_tr_building_cond__W_K_1") in
// one language, for people. CampaignID 0 is the default (set by whoever made the device type or
// energy query type); a label for a campaign overrides the default in that campaign. The unit is
// not part of the label: the app writes it from the property name.
type PropertyLabel struct {
	PropertyName string `json:"property"`
	CampaignID   uint   `json:"campaign_id,omitempty"`
	Language     string `json:"language"`
	// An empty label removes the label.
	Label string `json:"label"`
}

const (
	MaxPropertyName = 191
	MaxLabel        = 255
)

var (
	ErrPropertyName = errors.New("property name missing or too long")
	ErrLabel        = errors.New("label too long")
)

// Normalize trims the label and checks the property name; the language becomes "nl-NL" style
// (an empty language is campaign.DefaultLanguage).
func (l PropertyLabel) Normalize() (PropertyLabel, error) {
	l.PropertyName = strings.TrimSpace(l.PropertyName)
	if l.PropertyName == "" || len(l.PropertyName) > MaxPropertyName {
		return l, ErrPropertyName
	}
	l.Label = strings.TrimSpace(l.Label)
	if utf8.RuneCountInString(l.Label) > MaxLabel {
		return l, ErrLabel
	}
	language, err := campaign.NormalizeLanguage(l.Language)
	if err != nil {
		return l, err
	}
	l.Language = language
	return l, nil
}
