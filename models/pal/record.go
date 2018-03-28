package modelsPal

type Record struct {
	Id				string				`json:"id"`
	Fields			Fields				`json:"fields"`
	MasterFile		Masterfile			`json:"masterFile"`
}
