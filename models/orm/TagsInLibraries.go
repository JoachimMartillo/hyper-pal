package modelsOrm

import (
	"time"
	"hyper-pal/models/pal"
	"lib-go-logger/logger"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego"
)

type TagsInLibraries struct {
	Uuid			   		string			`orm:"pk"`
	CreatedAt				time.Time
	ModifiedAt				time.Time
	LibraryUuid				string
	Txt						string
	ParentId				*string
	IsResource				bool
}

func (*TagsInLibraries) TableName() string {
	return "TagsInLibraries"
}

func (o *TagsInLibraries) AddClassification(ormer orm.Ormer, classification *modelsPal.Classification, parentTagId *string, libraryUuid string) (*TagsInLibraries, error) {
	// Fill
	o.Uuid = uuid.NewV4String()
	o.CreatedAt = time.Now()
	o.ModifiedAt = o.CreatedAt
	o.LibraryUuid = libraryUuid
	o.Txt = classification.Name
	o.ParentId = parentTagId

	// Check maybe Resource in config
	configIsResource, _ := beego.AppConfig.Bool("hyper.importer.tag.isResource")
	o.IsResource = configIsResource


	// Save
	if _, err := ormer.Insert(o); err != nil {
		return nil, err
	}

	return o, nil
}
