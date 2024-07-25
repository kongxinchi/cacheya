package compressor

import (
	"bytes"
	"compress/zlib"
	"io"
)

type ZlibCompressor struct {
}

func NewZlibCompressor() *ZlibCompressor {
	return &ZlibCompressor{}
}

func (c *ZlibCompressor) Encode(b []byte) ([]byte, error) {
	var encodeBuf bytes.Buffer

	writer := zlib.NewWriter(&encodeBuf)
	if _, err := writer.Write(b); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return encodeBuf.Bytes(), nil
}

func (c *ZlibCompressor) Decode(b []byte) ([]byte, error) {
	var decodeBuf bytes.Buffer

	reader, err := zlib.NewReader(bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(&decodeBuf, reader); err != nil {
		return nil, err
	}
	if err = reader.Close(); err != nil {
		return nil, err
	}
	return decodeBuf.Bytes(), nil
}
