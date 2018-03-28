package modelsHyper

import (
	"hyper-pal/models/data"
	"path/filepath"
)

type ContentItem struct {
	Id					string				`json:"id"`
	Filename			string				`json:"filename"`
	Progress			int					`json:"progress"`
	ResourceTagsUuid	[]string			`json:"resourceTagsUuid"`
	Size				int64				`json:"size"`
	SubmittedBy			string				`json:"submittedBy"`
	Tags				[]string			`json:"tags"`
	TagsUuid			[]string			`json:"tagsUuid"`
	TemporaryFilePath	string				`json:"temporaryFilePath"`
	Title				string				`json:"title"`
	Description			string				`json:"description"`
}

func CreateContentItemFromFile(file *modelsData.File) *ContentItem {
	o := &ContentItem{
		Id: file.Id,
		Filename: file.Filename,
		Progress: 100,
		ResourceTagsUuid: make([]string, 0),
		Size: file.Size,
		Tags: make([]string, 0),
		TagsUuid: make([]string, 0),
	}

	return o.SetTitleFromFilename()
}

func (o *ContentItem) SetTitleFromFilename() *ContentItem {
	if o.Filename != "" {
		o.Title = filepath.Base(o.Filename)
	}
	return o
}