package measurement

import (
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat"
	"github.com/energiegedrag-nl/needforheat-server-api/needforheat/property"
)

// A Measurement is a measured value for a specific property.
type Measurement struct {
	ID         uint              `json:"id"`
	UploadID   uint              `json:"upload_id"`
	PropertyID int               `json:"-"`
	Property   property.Property `json:"property"`
	Time       needforheat.Time  `json:"time"`
	Value      string            `json:"value"`
}
