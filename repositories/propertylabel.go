package repositories

import (
	"time"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/propertylabel"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Database representation of a [propertylabel.PropertyLabel]: one row per property, campaign
// (0: the default) and language.
type PropertyLabelModel struct {
	ID           uint   `gorm:"primarykey"`
	PropertyName string `gorm:"size:191;not null;uniqueIndex:idx_property_label,priority:1"`
	CampaignID   uint   `gorm:"not null;default:0;uniqueIndex:idx_property_label,priority:2"`
	Language     string `gorm:"size:16;not null;uniqueIndex:idx_property_label,priority:3"`
	Label        string `gorm:"size:255;not null"`
	UpdatedAt    time.Time
}

func (PropertyLabelModel) TableName() string {
	return "property_label"
}

type PropertyLabelRepository struct {
	db *gorm.DB
}

func NewPropertyLabelRepository(db *gorm.DB) *PropertyLabelRepository {
	return &PropertyLabelRepository{db: db}
}

// Set stores the labels (normalized), replacing earlier ones for the same property, campaign and
// language; an empty label removes it. All or nothing.
func (r *PropertyLabelRepository) Set(labels []propertylabel.PropertyLabel) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, l := range labels {
			if l.Label == "" {
				err := tx.Where("property_name = ? AND campaign_id = ? AND language = ?",
					l.PropertyName, l.CampaignID, l.Language).Delete(&PropertyLabelModel{}).Error
				if err != nil {
					return err
				}
				continue
			}
			row := PropertyLabelModel{PropertyName: l.PropertyName, CampaignID: l.CampaignID, Language: l.Language, Label: l.Label}
			err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "property_name"}, {Name: "campaign_id"}, {Name: "language"}},
				DoUpdates: clause.AssignmentColumns([]string{"label", "updated_at"}),
			}).Create(&row).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Resolve gives the label per property name in one language for one campaign: the campaign's
// own label where it has one, else the default.
func (r *PropertyLabelRepository) Resolve(campaignID uint, language string) (map[string]string, error) {
	var rows []PropertyLabelModel
	err := r.db.Where("language = ? AND campaign_id IN ?", language, []uint{0, campaignID}).
		Order("campaign_id").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows { // defaults (campaign 0) first, then the campaign's own
		out[row.PropertyName] = row.Label
	}
	return out, nil
}
