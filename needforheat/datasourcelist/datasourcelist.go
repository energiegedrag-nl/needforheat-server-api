package datasourcelist

import "github.com/energiegedrag-nl/needforheat-server-api/needforheat/datasourcetype"

// A datasourcelist is a collection of datasourcetypes
type DataSourceList struct {
	ID    uint                            `json:"id"`
	Name  string                          `json:"name"`
	Items []datasourcetype.DataSourceType `json:"items"`
}

// Create a new DataSourceList.
func MakeDataSourceList(items []datasourcetype.DataSourceType, name string) DataSourceList {
	return DataSourceList{
		Items: items,
		Name:  name,
	}
}
