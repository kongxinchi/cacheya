package compressor

import "github.com/golang/snappy"

type SnappyCompressor struct {
}

func NewSnappyCompressor() *SnappyCompressor {
	return &SnappyCompressor{}
}

func (s *SnappyCompressor) Encode(bytes []byte) ([]byte, error) {
	return snappy.Encode(nil, bytes), nil
}

func (s *SnappyCompressor) Decode(bytes []byte) ([]byte, error) {
	return snappy.Decode(nil, bytes)
}
