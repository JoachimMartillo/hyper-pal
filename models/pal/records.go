package modelsPal

// for handling array of records.
type Records struct {
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`
	TotalCount int      `json:"totalCount"`
	Items      []Record `json:"items"`
}
