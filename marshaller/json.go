package marshaller

import json "github.com/json-iterator/go"

type JsonMarshaller struct {
}

func NewJsonMarshaller() *JsonMarshaller {
	return &JsonMarshaller{}
}

func (j *JsonMarshaller) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JsonMarshaller) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
