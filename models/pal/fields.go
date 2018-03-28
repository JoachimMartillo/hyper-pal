package modelsPal

type Fields struct {
	Items		[]Field				`json:"items"`
}

func (o *Fields) FindByFieldName(fieldName string) *Field {
	for _, field := range o.Items {
		if field.FieldName == fieldName {
			return &field
		}
	}
	return nil
}
