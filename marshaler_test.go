package cacheya

import (
	"fmt"
	"testing"
	"time"

	"github.com/kongxinchi/cacheya/marshaller"
)

type MarshallerTestObject struct {
	ID               int64
	State            string
	QwExternalUserID string
	UID              string
	ContactID        *string
}

func TestMarshaller_Benchmark(t *testing.T) {
	marshallerBenchmark("JSON", marshaller.NewJsonMarshaller())
	marshallerBenchmark("MSGPACK", marshaller.NewMsgpackMarshaller())
}

func marshallerBenchmark(name string, marshaller Marshaller) {
	contactID := "64b4dcf8-e0c9-11ee-8c00-dafebafdd11e"
	o := &MarshallerTestObject{
		ID:               258497503316467713,
		State:            "active",
		QwExternalUserID: "wmJawBCQAA7R7dpI98pWhYWzWstkI86A",
		UID:              "6",
		ContactID:        &contactID,
	}

	begin := time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = marshaller.Marshal(o)
	}
	fmt.Printf("%s Marshal: %s\n", name, time.Since(begin))

	s, _ := marshaller.Marshal(o)
	begin = time.Now()
	for i := 0; i < 10000; i++ {
		_ = marshaller.Unmarshal(s, o)
	}
	fmt.Printf("%s Unmarshal: %s\n", name, time.Since(begin))
}
