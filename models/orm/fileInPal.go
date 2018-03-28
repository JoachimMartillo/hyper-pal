package modelsOrm

import (
	"time"
	"hyper-pal/models/data"
)

type FileInPal struct {
	ContentItemId   string			`orm:"pk"`
	SpaceId			string
	RecordId        string
	FileId			string
	Filename		string
	Size			int64
	Description		string
	CreatedAt		time.Time
	ModifiedAt		time.Time
}

func (*FileInPal) TableName() string {
	return "filesInPal"
}

func CreateFileInPal(contentItemId, spaceId, recordId string, file *modelsData.File) *FileInPal {
	return &FileInPal{
		contentItemId,
		spaceId,
		recordId,
		file.ExternalId,
		file.Filename,
		file.Size,
		file.Description,
		time.Now(),
		time.Now(),
	}
}
