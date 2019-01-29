package modelsPal

// something related to an OrderFile
type BodyFileOrder struct {
	Type                string   `json:"type"`
	DisableNotification string   `json:"disableNotification"`
	Priority            int      `json:"priority"`
	SiteAffinity        string   `json:"siteAffinity"`
	SatelliteId         *string  `json:"satelliteId"`
	Targets             []Target `json:"targets"`
}

func CreateBodyFileOrder() *BodyFileOrder {
	return &BodyFileOrder{
		Type:                "download",
		DisableNotification: "true",
		Priority:            0,
		SiteAffinity:        "DEFAULT",
		SatelliteId:         nil,
		Targets:             make([]Target, 0),
	}
}

func (o *BodyFileOrder) AddTargetRecordId(recordId string) *BodyFileOrder {
	o.Targets = append(o.Targets, Target{
		RecordId:    recordId,
		TargetTypes: []string{"Document"},
		AssetType:   "LatestVersionOfMasterFile",
	})
	return o
}
