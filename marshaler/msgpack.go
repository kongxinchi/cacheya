package marshaler

import "github.com/vmihailenco/msgpack/v5"

type MsgpackMarshaler struct {
}

func NewMsgpackMarshaler() *MsgpackMarshaler {
	return &MsgpackMarshaler{}
}

func (m *MsgpackMarshaler) Marshal(v any) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (m *MsgpackMarshaler) Unmarshal(data []byte, v any) error {
	return msgpack.Unmarshal(data, v)
}
