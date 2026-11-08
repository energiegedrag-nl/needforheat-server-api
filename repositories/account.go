// Package repositories implements repositories for use in services.
package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/account"
	"gorm.io/gorm"
)

type AccountRepository struct {
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

// Database representation of a [account.Account].
type AccountModel struct {
	gorm.Model
	CampaignModelID uint `gorm:"column:campaign_id"`
	Campaign        CampaignModel
	ActivatedAt     *needforheat.Time
	CloudFeeds      []CloudFeedModel `gorm:"foreignKey:AccountID"`
	Devices         []DeviceModel
}

// Set the name of the table in the database.
func (AccountModel) TableName() string {
	return "account"
}

// Create a new AccountModel from a [account.Account].
func MakeAccountModel(account account.Account) AccountModel {
	return AccountModel{
		Model: gorm.Model{
			ID: account.ID,
		},
		CampaignModelID: account.Campaign.ID,
		Campaign:        MakeCampaignModel(account.Campaign),
		ActivatedAt:     account.ActivatedAt,
	}
}

// Create a [account.Account] from an AccountModel.
func (m *AccountModel) fromModel() account.Account {
	return account.Account{
		ID:          m.Model.ID,
		Campaign:    m.Campaign.fromModel(),
		ActivatedAt: m.ActivatedAt,
	}
}

func (r *AccountRepository) Find(accountToFind account.Account) (account.Account, error) {
	accountModel := MakeAccountModel(accountToFind)
	err := r.db.Preload("Campaign.App").Where(&accountModel).First(&accountModel).Error

	var campaignModel CampaignModel
	errCm := r.db.Where("id = ?", accountModel.Campaign.ID).First(&campaignModel).Error
	if errCm != nil {
		return account.Account{}, err
	}

	var dataSourceList DataSourceListModel
	dsErr := r.db.Preload("Items").Where("id = ?", campaignModel.DataSourceListID).First(&dataSourceList).Error
	if dsErr != nil {
		return account.Account{}, dsErr
	}

	accountAPI := accountModel.fromModel()
	accountAPI.Campaign.DataSourceList = dataSourceList.fromModel(r.db)

	return accountAPI, err
}

func (r *AccountRepository) GetAll() ([]account.Account, error) {
	accounts := make([]account.Account, 0)

	var accountModels []AccountModel
	err := r.db.Preload("Campaign.App").Find(&accountModels).Error
	if err != nil {
		return nil, err
	}

	for _, accountModel := range accountModels {
		var campaignModel CampaignModel
		errCm := r.db.Where("id = ?", accountModel.Campaign.ID).First(&campaignModel).Error
		if errCm != nil {
			return nil, err
		}

		var dataSourceList DataSourceListModel
		dsErr := r.db.Preload("Items").Where("id = ?", campaignModel.DataSourceListID).First(&dataSourceList).Error
		if dsErr != nil {
			return nil, dsErr
		}

		accountAPI := accountModel.fromModel()
		accountAPI.Campaign.DataSourceList = dataSourceList.fromModel(r.db)

		accounts = append(accounts, accountAPI)
	}

	return accounts, nil
}

func (r *AccountRepository) Create(account account.Account) (account.Account, error) {
	accountModel := MakeAccountModel(account)
	err := r.db.Create(&accountModel).Error
	return accountModel.fromModel(), err
}

func (r *AccountRepository) Update(account account.Account) (account.Account, error) {
	accountModel := MakeAccountModel(account)
	err := r.db.Model(&accountModel).Updates(accountModel).Error
	return accountModel.fromModel(), err
}

func (r *AccountRepository) Delete(account account.Account) error {
	accountModel := MakeAccountModel(account)
	return r.db.Delete(&accountModel).Error
}
