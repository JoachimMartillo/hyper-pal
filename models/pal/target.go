package modelsPal

// something from Adam RestAPI
type Target struct {
	RecordId    string   `json:"recordId"`
	TargetTypes []string `json:"targetTypes"`
	AssetType   string   `json:"assetType"`
}
