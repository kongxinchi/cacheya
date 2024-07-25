package cacheya

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/kongxinchi/cacheya/compressor"
	"github.com/stretchr/testify/assert"
)

type CompressorTestObject struct {
	ID               int64
	State            string
	QwExternalUserID string
	UID              string
	ContactID        *string
}

func TestCompressor(t *testing.T) {
	compressorCorrectness(t, "Noop", compressor.NewNoopCompressor())
	compressorCorrectness(t, "Snappy", compressor.NewSnappyCompressor())
	compressorCorrectness(t, "Zlib", compressor.NewZlibCompressor())
}

func compressorCorrectness(t *testing.T, name string, compressor Compressor) {
	contactID := "64b4dcf8-e0c9-11ee-8c00-dafebafdd11e"
	o := &CompressorTestObject{
		ID:               258497503316467713,
		State:            "active",
		QwExternalUserID: "wmJawBCQAA7R7dpI98pWhYWzWstkI86A",
		UID:              "6",
		ContactID:        &contactID,
	}
	text, _ := json.Marshal(o)
	text = []byte(fmt.Sprintf("%s%s", text, text))

	encoded, err := compressor.Encode(text)
	assert.Nil(t, err)

	decoded, err := compressor.Decode(encoded)
	assert.Nil(t, err)

	assert.Equal(t, text, decoded)
}

func TestCompressor_Benchmark(t *testing.T) {
	compressorBenchmark("Noop", compressor.NewNoopCompressor())
	compressorBenchmark("Snappy", compressor.NewSnappyCompressor())
	compressorBenchmark("Zlib", compressor.NewZlibCompressor())
}

func compressorBenchmark(name string, compressor Compressor) {
	contactID := "64b4dcf8-e0c9-11ee-8c00-dafebafdd11e"
	o := &CompressorTestObject{
		ID:               258497503316467713,
		State:            "active",
		QwExternalUserID: "wmJawBCQAA7R7dpI98pWhYWzWstkI86A",
		UID:              "6",
		ContactID:        &contactID,
	}
	text, _ := json.Marshal(o)
	text = []byte(fmt.Sprintf("%s%s", text, text))

	begin := time.Now()
	var s []byte
	for i := 0; i < 10000; i++ {
		s, _ = compressor.Encode(text)
	}
	fmt.Printf("%s Encode: %s, Len: %d\n", name, time.Since(begin), len(s))

	begin = time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = compressor.Decode(s)
	}
	fmt.Printf("%s Dencode: %s\n", name, time.Since(begin))
}
