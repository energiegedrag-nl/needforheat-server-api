package repositories

import (
	"fmt"

	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcelist"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcetype"
	"gorm.io/gorm"
)

type DataSourceListRepository struct {
	db *gorm.DB
}

func NewDataSourceListRepository(db *gorm.DB) *DataSourceListRepository {
	return &DataSourceListRepository{
		db: db,
	}
}

// Database representation of a [datasourcelist.DataSourceList].
type DataSourceListModel struct {
	gorm.Model
	Items    []DataSourceListItems
	Campaign []CampaignModel `gorm:"foreignKey:DataSourceListID"`
	Name     string
}

// Set the name of the table in the database.
func (DataSourceListModel) TableName() string {
	return "data_source_list"
}

// Create a new DataSourceListModel from a [datasourcelist.DataSourceList]
func MakeDataSourceListModel(dataSourceList datasourcelist.DataSourceList) DataSourceListModel {
	var dataSourceListItems []DataSourceListItems

	for _, item := range dataSourceList.Items {
		dataSourceListItems = append(dataSourceListItems, DataSourceListItems{
			DataSourceListModelID: dataSourceList.ID,
			DataSourceTypeModelID: item.ID,
			Order:                 item.Order,
		})
	}

	return DataSourceListModel{
		Model: gorm.Model{ID: dataSourceList.ID},
		Name:  dataSourceList.Name,
		Items: dataSourceListItems,
	}
}

// Create a [datasourcelist.DataSourceList] from a DataSourceListModel.
func (m *DataSourceListModel) fromModel(db *gorm.DB) datasourcelist.DataSourceList {
	var dataSourceListItems []datasourcetype.DataSourceType

	// Initialize DataSourceList object
	dataSourceList := datasourcelist.DataSourceList{
		ID:   m.Model.ID,
		Name: m.Name,
	}

	for _, item := range m.Items {
		var dataSourceType DataSourceTypeModel

		// Fetch DataSourceTypeModel and Order using JOIN and Preload
		if err := db.
			Preload("Precedes").
			First(&dataSourceType, item.DataSourceTypeModelID).
			Error; err != nil {
			// Handle error, e.g., log the error or return empty list
			fmt.Printf("Error fetching DataSourceType for item %d: %s\n", item.ID, err)
			continue // Skip this item and proceed with the next one
		}

		var dataSourceListItem DataSourceListItems
		if err := db.
			Where("data_source_list_model_id = ? AND data_source_type_model_id = ?", m.Model.ID, item.DataSourceTypeModelID).
			First(&dataSourceListItem).
			Error; err != nil {
			fmt.Printf("Error fetching DataSourceListItems for item %d: %s\n", item.ID, err)
			continue // Skip this item and proceed with the next one
		}

		// Convert DataSourceTypeModel to DataSourceType and append to the list
		dataSourceTypeModel := dataSourceType.fromModel(db)
		dataSourceTypeModel.Order = dataSourceListItem.Order
		dataSourceListItems = append(dataSourceListItems, dataSourceTypeModel)
	}

	// Set the populated Items list to the DataSourceList object
	dataSourceList.Items = dataSourceListItems

	return dataSourceList
}

func (r *DataSourceListRepository) Create(dataSourceList datasourcelist.DataSourceList) (datasourcelist.DataSourceList, error) {
	// Check for duplicate orders
	orderMap := make(map[uint]bool)
	orderMap[0] = true //Gorm decoder makes it always 0, making a custom decoder to make it -1 should be done in the future
	for _, item := range dataSourceList.Items {
		if orderMap[item.Order] && item.Order != 0 {
			return datasourcelist.DataSourceList{}, fmt.Errorf("duplicate order found: %d", item.Order)
		}
		orderMap[item.Order] = true
	}
	tx := r.db.Begin()

	// Defer rollback in case of error
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create DataSourceListModel instance
	dataSourceListModel := MakeDataSourceListModel(datasourcelist.DataSourceList{Name: dataSourceList.Name})
	if err := tx.Create(&dataSourceListModel).Error; err != nil {
		tx.Rollback()
		return datasourcelist.DataSourceList{}, err
	}
	dataSourceListModel = MakeDataSourceListModel(datasourcelist.DataSourceList{ID: dataSourceListModel.ID, Name: dataSourceListModel.Name, Items: dataSourceList.Items})

	// Create DataSourceListItems (relationship between DataSourceListModel and DataSourceTypeModel)
	for _, item := range dataSourceList.Items {
		orderNumber, _ := findMaxKey(orderMap)

		if item.Order == 0 {
			orderNumber++
			orderMap[orderNumber] = true
			item.Order = orderNumber
		} else {
			orderNumber = item.Order
		}

		dataSourceListItems := DataSourceListItems{
			DataSourceListModelID: dataSourceListModel.ID,
			DataSourceTypeModelID: item.ID,
			Order:                 orderNumber,
		}
		if err := tx.Create(&dataSourceListItems).Error; err != nil {
			tx.Rollback()
			return datasourcelist.DataSourceList{}, err
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return datasourcelist.DataSourceList{}, err
	}

	return dataSourceListModel.fromModel(r.db), nil
}

func (r *DataSourceListRepository) Delete(datasourcelist datasourcelist.DataSourceList) error {
	datasourceListModel := MakeDataSourceListModel(datasourcelist)
	return r.db.Create(&datasourceListModel).Error
}

func (r *DataSourceListRepository) Find(datasourceList datasourcelist.DataSourceList) (datasourcelist.DataSourceList, error) {
	datasourceListModel := MakeDataSourceListModel(datasourceList)
	err := r.db.Preload("Items").Where(&datasourceListModel).First(&datasourceListModel).Error
	return datasourceListModel.fromModel(r.db), err
}

func (r *DataSourceListRepository) GetAll() ([]datasourcelist.DataSourceList, error) {
	var datasourceLists []datasourcelist.DataSourceList

	var datasourceListsModels []DataSourceListModel
	err := r.db.Preload("Items").Find(&datasourceListsModels).Error
	if err != nil {
		return nil, err
	}

	for _, datasourceListModel := range datasourceListsModels {
		datasourceLists = append(datasourceLists, datasourceListModel.fromModel(r.db))
	}

	return datasourceLists, nil
}

func findMaxKey(orderMap map[uint]bool) (uint, error) {
	if len(orderMap) == 0 {
		return 0, fmt.Errorf("map is empty")
	}

	var maxKey uint

	for key := range orderMap {
		if key > maxKey {
			maxKey = key
		}
	}

	return maxKey, nil
}
