package marshaler

import json "github.com/json-iterator/go"

type JsonMarshaler struct {
}

func NewJsonMarshaler() *JsonMarshaler {
	return &JsonMarshaler{}
}

func (j *JsonMarshaler) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JsonMarshaler) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
