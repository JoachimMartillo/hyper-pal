package modelsPal

const ORDER_STATUS_SUCCESS = "Success"
const ORDER_STATUS_EXECUTING = "Executing"

// what is an order file???
type OrderFile struct {
	Id             string   `json:"id"`
	Status         string   `json:"status"`
	Message        string   `json:"message"`
	ExecutionTime  string   `josn:"executionTime"`
	DeliveredFiles []string `json:"deliveredFiles"`
}

func (o *OrderFile) GetFirstFileLink() (link string) {
	for _, deliveredFile := range o.DeliveredFiles {
		if deliveredFile != "" {
			link = deliveredFile
			break
		}
	}
	return
}
