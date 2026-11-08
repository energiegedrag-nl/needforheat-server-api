package services

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
)

type PropertyService struct {
	repository property.PropertyRepository
}

// Create a new PropertyService.
func NewPropertyService(repository property.PropertyRepository) *PropertyService {
	return &PropertyService{
		repository: repository,
	}
}

func (s *PropertyService) Create(name string) (property.Property, error) {
	property := property.MakeProperty(name)
	return s.repository.Create(property)
}

func (s *PropertyService) Find(property property.Property) (property.Property, error) {
	return s.repository.Find(property)
}

func (s *PropertyService) GetByID(id uint) (property.Property, error) {
	return s.repository.Find(property.Property{ID: id})
}

func (s *PropertyService) GetByName(name string) (property.Property, error) {
	return s.repository.Find(property.Property{Name: name})
}
