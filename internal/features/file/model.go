package file

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type File struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	OriginalFilename string             `json:"original_filename" bson:"original_filename"`
	URL              string             `json:"url" bson:"url"`
	Path             string             `json:"path" bson:"path"`
	Size             int64              `json:"size" bson:"size"`
	MimeType         string             `json:"mime_type" bson:"mime_type"`
	ModuleName       string             `json:"module_name,omitempty" bson:"module_name,omitempty"`
	RecordID         string             `json:"record_id,omitempty" bson:"record_id,omitempty"`
	UploadedBy       primitive.ObjectID `json:"uploaded_by" bson:"uploaded_by"`
	IsShared         bool               `json:"is_shared" bson:"is_shared"`
	StorageType      string             `json:"storage_type" bson:"storage_type"` // local, s3, etc.
	Description      string             `json:"description,omitempty" bson:"description,omitempty"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
}
