package modelsOrm

import (
	"time"
	"hyper-pal/models/data"
	"github.com/astaxie/beego/orm"
	"hyper-pal/models/pal"
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
	return "FilesInPal"
}

func (o *FileInPal) HadChanged(file *modelsPal.File) bool {
	return !(o.Title == file.Title && o.Description == file.Description && o.Filename == file.FileName && o.Size == file.FileSize)
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

func (o *FileInPal) UpdateByFile(ormer orm.Ormer, file *modelsPal.File, newContentItemId string) (contentItemId string, err error) {
	contentItemId = o.ContentItemId

	o.Title = file.Title
	o.Description = file.Description
	o.Filename = file.FileName
	o.Size = file.FileSize
	o.ModifiedAt = time.Now()

	if newContentItemId != "" && newContentItemId != o.ContentItemId {
		/*
		// Change PK
		_, err = ormer.QueryTable(o).Filter("content_item_id", o.ContentItemId).Update(orm.Params{
			"title":           o.Title,
			"description":     o.Description,
			"filename":        o.Filename,
			"size":            o.Size,
			"content_item_id": newContentItemId,
			"modified_at":	   o.ModifiedAt,
		})
		*/
		err = ormer.Begin()
		_, err = ormer.Raw("update ContentItem set uuid = ? where uuid = ?", contentItemId, newContentItemId).Exec()
		if err == nil {
			_, err = ormer.Update(o, "title", "description", "filename", "size", "modified_at")
		}
		if err == nil {
			err = ormer.Commit()
		} else {
			ormer.Rollback()
		}
	} else {
		_, err = ormer.Update(o, "title", "description", "filename", "size", "modified_at")
	}

	return
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
