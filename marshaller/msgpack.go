package marshaller

import "github.com/vmihailenco/msgpack/v5"

type MsgpackMarshaller struct {
}

func NewMsgpackMarshaller() *MsgpackMarshaller {
	return &MsgpackMarshaller{}
}

func (m *MsgpackMarshaller) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (m *MsgpackMarshaller) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}
