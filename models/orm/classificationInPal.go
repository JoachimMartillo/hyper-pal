package modelsOrm

import (
	"time"
	"github.com/astaxie/beego/orm"
	"hyper-pal/models/pal"
)

type ClassificationInPal struct {
	ClassificationId   		string			`orm:"pk"`
	ParentClassificationId	*string
	TagId					*string
	CreatedAt				time.Time
}

func (*ClassificationInPal) TableName() string {
	return "classifications_in_pal"
}

func (o *ClassificationInPal) LoadByClassificationId(ormer orm.Ormer, classificationId string) (*ClassificationInPal, error) {
	o.ClassificationId = classificationId
	if err := ormer.Read(o, "ClassificationId"); err != nil {
		if err == orm.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

func (o *ClassificationInPal) Insert(ormer orm.Ormer, classification *modelsPal.Classification, parentTagId *string, topClassificationId, libraryUuid string) (*ClassificationInPal, error) {
	err := ormer.Begin()
	if err != nil {
		return o, err
	}
	var tagId *string
	if parentTagId == nil && classification.ParentId != topClassificationId {
		tagId = nil
	} else {
		// Insert tag first.
		var tag *TagsInLibraries
		tag, err = new(TagsInLibraries).AddClassification(ormer, classification, parentTagId, libraryUuid)
		if err != nil {
			ormer.Rollback()
			return o, err
		}
		tagId =  &tag.Uuid
	}
	// Insert obj.
	if _, err = ormer.Insert(o.FillFromClassification(classification, tagId)); err != nil {
		ormer.Rollback()
		return o, err
	}
	if err = ormer.Commit(); err != nil {
		ormer.Rollback()
		return o, err
	}
	return o, nil
}

func (o *ClassificationInPal) FillFromClassification(classification *modelsPal.Classification, tagId *string) *ClassificationInPal {
	o.ClassificationId = classification.Id
	o.TagId = tagId
	o.CreatedAt = time.Now()
	if classification.ParentId == "" {
		o.ParentClassificationId = nil
	} else {
		o.ParentClassificationId = &classification.ParentId
	}
	return o
}
