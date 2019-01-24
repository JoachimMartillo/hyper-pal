package modelsData

import (
	"pal-importer/models/pal"
	"pal-importer/system"
)

type File struct {
	Id          string
	ExternalId  string
	Filename    string
	Size        int64
	Description string
	Title       string
	Fullpath    string
}

func CreateFileFromPal(palFile *modelsPal.File) *File {
	return &File{
		Id:          system.NewV4String(),
		ExternalId:  palFile.Id,
		Filename:    palFile.FileName,
		Size:        palFile.FileSize,
		Description: palFile.Description,
		Title:       palFile.Title,
		Fullpath:    palFile.OutFilename,
	}
}
