package modelsPal

const STATUS_SUCCESS = "Success"

type OrderFile struct {
	Id				string				`json:"id"`
	Status			string				`json:"status"`
	Message			string				`json:"message"`
	DeliveredFiles	[]string			`json:"deliveredFiles"`
}
