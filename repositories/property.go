package repositories

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
	"gorm.io/gorm"
)

type PropertyRepository struct {
	db *gorm.DB
}

// Create a new PropertyRepository.
func NewPropertyRepository(db *gorm.DB) *PropertyRepository {
	return &PropertyRepository{
		db: db,
	}
}

// Database representation of a [property.Property]
type PropertyModel struct {
	gorm.Model
	Name string `gorm:"unique;non null"`
}

// Set the name of the table in the database.
func (PropertyModel) TableName() string {
	return "property"
}

// Create a PropertyModel from a [property.Property].
func MakePropertyModel(property property.Property) PropertyModel {
	return PropertyModel{
		Model: gorm.Model{ID: property.ID},
		Name:  property.Name,
	}
}

// Create a [property.Property] from a PropertyModel.
func (m *PropertyModel) fromModel() property.Property {
	return property.Property{
		ID:   m.Model.ID,
		Name: m.Name,
	}
}

func (r *PropertyRepository) Find(property property.Property) (property.Property, error) {
	propertyModel := MakePropertyModel(property)
	err := r.db.Where(&propertyModel).First(&propertyModel).Error
	return propertyModel.fromModel(), err
}

func (r *PropertyRepository) GetAll() ([]property.Property, error) {
	var properties []property.Property

	var propertyModels []PropertyModel
	err := r.db.Find(&propertyModels).Error
	if err != nil {
		return nil, err
	}

	for _, propertyModel := range propertyModels {
		properties = append(properties, propertyModel.fromModel())
	}

	return properties, nil
}

func (r *PropertyRepository) Create(property property.Property) (property.Property, error) {
	propertyModel := MakePropertyModel(property)
	err := r.db.Create(&propertyModel).Error
	return propertyModel.fromModel(), err
}

func (r *PropertyRepository) Delete(property property.Property) error {
	propertyModel := MakePropertyModel(property)
	return r.db.Delete(&propertyModel).Error
}
