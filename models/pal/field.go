package modelsPal

type Field struct {
	Id				string				`json:"id"`
	FieldName		string				`json:"fieldName"`
	Label			string				`json:"label"`
	LocalizedValues []LocalizedValue	`json:"localizedValues"`
}

func (o *Field) GetValueFirstString() string {
	first := o.getValueFirst()
	if first == nil {
		return ""
	} else {
		return o.getValueFirst().(string)
	}
}

func (o *Field) getValueFirst() interface{} {
	var result interface{}
	for _, localizedValue := range o.LocalizedValues {
		result = localizedValue.Value
		if result == nil {
			for _, value := range localizedValue.Values {
				result = value
				if result != nil {
					break
				}
			}
		}
		if result != nil {
			break
		}
	}
	return result
}
