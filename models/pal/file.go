package modelsPal

type File struct {
	Id				string				`json:"id"`
	FileName		string				`json:"fileName"`
	FileSize		int64				`json:"fileSize"`

	Description		string				`json:"-"`	// We just store description from other object here.
	OutFilename		string				`json:"-"`	// Filename in OS where we stored downloaded file.
}
