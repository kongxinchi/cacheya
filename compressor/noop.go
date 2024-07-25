package compressor

type NoopCompressor struct {
}

func NewNoopCompressor() *NoopCompressor {
	return &NoopCompressor{}
}

func (n *NoopCompressor) Encode(bytes []byte) ([]byte, error) {
	return bytes, nil
}

func (n *NoopCompressor) Decode(bytes []byte) ([]byte, error) {
	return bytes, nil
}
