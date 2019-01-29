package modelsPal

// these are go interfaces and not hyper cms interfaces
type LocalizedValue struct {
	Value  interface{}   `json:"value"`
	Values []interface{} `json:"values"`
}
