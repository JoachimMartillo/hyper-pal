package modelsData

import (
	"hyper-pal/models/pal"
	"github.com/satori/go.uuid"
)

type File struct {
	Id				string
	ExternalId		string
	Filename		string
	Size			int64
	Description		string
	Fullpath		string
}

func CreateFileFromPal(palFile *modelsPal.File) *File {
	return &File{
		Id: 			uuid.NewV4().String(),
		ExternalId:		palFile.Id,
		Filename:		palFile.FileName,
		Size:			int64(palFile.FileSize),
		Description:	palFile.Description,
		Fullpath:		palFile.OutFilename,
	}
}
