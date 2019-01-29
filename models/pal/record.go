package modelsPal

// the JSON description of a Record
type Record struct {
	Id             string                `json:"id"`
	Fields         Fields                `json:"fields"`
	MasterFile     Masterfile            `json:"masterFile"`
	Classfications RecordClassifications `json:"classifications"`
}

type RecordClassifications struct {
	Items []RecordClassification `json:"items"`
}

type RecordClassification struct {
	Id string `json:"id"`
}
