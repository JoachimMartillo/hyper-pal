package modelsHyper

type LibraryRequest struct {
	LibraryTitle		string				`json:"libraryTitle"`
	LanguageUuid		string				`json:"languageUuid"`
	LanguageIdentifier	string				`json:"languageIdentifier"`
	LanguageName		string				`json:"languageName"`
}

func CreateLibraryRequest(title, languageUuid, languageIdentifier, languageName string) *LibraryRequest {
	return &LibraryRequest{
		LibraryTitle: 		title,
		LanguageUuid: 		languageUuid,
		LanguageIdentifier: languageIdentifier,
		LanguageName: 		languageName,
	}
}
