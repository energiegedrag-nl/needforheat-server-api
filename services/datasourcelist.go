package services

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcelist"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcetype"
)

type DataSourceListService struct {
	repository datasourcelist.DataSourceListRepository

	// Service used for the items
	shoppingListItemService *DataSourceTypeService
}

// Create a new DataSourceListService.
func NewDataSourceListService(repository datasourcelist.DataSourceListRepository, shoppingListItemService *DataSourceTypeService) *DataSourceListService {
	return &DataSourceListService{
		repository:              repository,
		shoppingListItemService: shoppingListItemService,
	}
}

func (s *DataSourceListService) Create(name string, items []datasourcetype.DataSourceType) (datasourcelist.DataSourceList, error) {
	var dataSourceListItems []datasourcetype.DataSourceType
	for _, item := range items {
		listItem, err := s.shoppingListItemService.Find(item)
		if err != nil {
			return datasourcelist.DataSourceList{}, err
		}
		listItem.Order = item.Order
		dataSourceListItems = append(dataSourceListItems, listItem)
	}
	datasourcelist := datasourcelist.MakeDataSourceList(dataSourceListItems, name)
	return s.repository.Create(datasourcelist)
}

func (s *DataSourceListService) Find(shoppingList datasourcelist.DataSourceList) (datasourcelist.DataSourceList, error) {
	return s.repository.Find(shoppingList)
}

func (s *DataSourceListService) GetAll() ([]datasourcelist.DataSourceList, error) {
	return s.repository.GetAll()
}

func (s *DataSourceListService) Delete(shoppingList datasourcelist.DataSourceList) error {
	return s.repository.Delete(shoppingList)
}
