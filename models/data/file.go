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
	Title			string
	Fullpath		string
}

func CreateFileFromPal(palFile *modelsPal.File) *File {
	return &File{
		Id: 			uuid.NewV4().String(),
		ExternalId:		palFile.Id,
		Filename:		palFile.FileName,
		Size:			palFile.FileSize,
		Description:	palFile.Description,
		Title:			palFile.Title,
		Fullpath:		palFile.OutFilename,
	}
}
