package modelsOrm

import (
	"github.com/astaxie/beego/orm"
	"fmt"
	"time"
)

const SPACE_STATUS_NEW				= 0
const SPACE_STATUS_ERROR			= 1
const SPACE_STATUS_TOPROCEED		= 2
const SPACE_STATUS_STARTED			= 3
const SPACE_STATUS_COMPLITED		= 4

type PalSpace struct {
	Uuid             string			`orm:"pk"`
	Name             string
	CompanyId        string
	LibraryId        string
	ClassificationId string
	Status           int8
	Message          string
	CreatedAt		 time.Time
	ModifiedAt		 time.Time
	ProceededAt		 time.Time
}

func (*PalSpace) TableName() string {
	return "PalSpaces"
}

func FindPalSpacesAll(ormer orm.Ormer) (spaces []*PalSpace, err error) {
	_, err = ormer.Raw(fmt.Sprintf("select * from %s", new(PalSpace).TableName())).QueryRows(&spaces)
	if err == orm.ErrNoRows {
		err = nil
	}
	return
}

func (o *PalSpace) UpdateProceededAt(ormer orm.Ormer) (err error) {
	_, err = ormer.Raw("update " + o.TableName() + " set proceeded_at = NOW() where uuid = ?", o.Uuid).Exec()
	return
}

func (o *PalSpace) UpdateStatus(ormer orm.Ormer, status int8, message string) (err error) {
	o.Status = status
	o.Message = message
	_, err = ormer.Raw("update " + o.TableName() + " set status = ?, message = ? where uuid = ?", status, message, o.Uuid).Exec()
	return
}

func (o *PalSpace) SetLibraryId(ormer orm.Ormer, libraryId string) (err error) {
	o.LibraryId = libraryId
	_, err = ormer.Raw("update " + o.TableName() + " set library_id = ? where uuid = ?", o.LibraryId, o.Uuid).Exec()
	return
}
