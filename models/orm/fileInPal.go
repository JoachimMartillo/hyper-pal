package modelsOrm

import (
	"time"
	"hyper-pal/models/data"
	"github.com/astaxie/beego/orm"
)

type FileInPal struct {
	ContentItemId   string			`orm:"pk"`
	LibraryId		string
	SpaceId			string
	RecordId        string
	FileId			string
	Filename		string
	Size			int64
	Title			string
	Description		string
	CreatedAt		time.Time
	ModifiedAt		time.Time
}

func (*FileInPal) TableName() string {
	return "files_in_pal"
}

func CreateFileInPal(contentItemId, libraryId, spaceId, recordId string, file *modelsData.File) *FileInPal {
	return &FileInPal{
		contentItemId,
		libraryId,
		spaceId,
		recordId,
		file.ExternalId,
		file.Filename,
		file.Size,
		file.Title,
		file.Description,
		time.Now(),
		time.Now(),
	}
}

// Can return nil if fileInPal is not imported.
func FindOneFileInPalByFirstUpload(ormer orm.Ormer, spaceId, recordId, libraryId string) (fileInPal *FileInPal, err error) {
	fileInPal = &FileInPal{
		SpaceId: spaceId,
		RecordId: recordId,
		LibraryId: libraryId,
	}
	if err = ormer.Read(fileInPal, "spaceId", "recordId", "libraryId"); err == orm.ErrNoRows {
		fileInPal = nil
		err = nil
	}
	return
}
