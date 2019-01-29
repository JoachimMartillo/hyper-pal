package modelsOrm

import (
	"github.com/astaxie/beego/orm"
	"pal-importer/models/data"
	"pal-importer/models/pal"
	"time"
)

type FileInPal struct {
	ContentItemId string `orm:"pk"`
	LibraryId     string
	SpaceId       string
	RecordId      string
	FileId        string
	Filename      string
	Size          int64
	Title         string
	Description   string
	CreatedAt     time.Time
	ModifiedAt    time.Time
}

// I wonder why the name of the FilesInPal datatable must be obtained by a method.
func (*FileInPal) TableName() string {
	return "FilesInPal"
}

// I hate not parenthesizing expressions like the one below -- but I resolved not to
// to modify the code until the files are fully commented.
func (o *FileInPal) HadChanged(file *modelsPal.File) bool {
	return !(o.Title == file.Title && o.Description == file.Description && o.Filename == file.FileName && o.Size == file.FileSize)
}

// most of the FileInPal record
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

// why is the primary key changed -- essentially unlinking item from other data tables
func (o *FileInPal) UpdateByFile(ormer orm.Ormer, file *modelsPal.File, newContentItemId string) (contentItemId string, err error) {
	contentItemId = o.ContentItemId

	o.Title = file.Title
	o.Description = file.Description
	o.Filename = file.FileName
	o.Size = file.FileSize
	o.ModifiedAt = time.Now()

	// should parenthesize
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
		SpaceId:   spaceId,
		RecordId:  recordId,
		LibraryId: libraryId,
	}
	if err = ormer.Read(fileInPal, "spaceId", "recordId", "libraryId"); err == orm.ErrNoRows {
		fileInPal = nil
		err = nil
	}
	return
}
