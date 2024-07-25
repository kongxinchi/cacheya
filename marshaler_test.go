package cacheya

import (
	"fmt"
	"testing"
	"time"

	"github.com/kongxinchi/cacheya/marshaler"
)

type MarshalerTestObject struct {
	ID               int64
	State            string
	QwExternalUserID string
	UID              string
	ContactID        *string
}

func TestMarshaler_Benchmark(t *testing.T) {
	marshalerBenchmark("JSON", marshaler.NewJsonMarshaler())
	marshalerBenchmark("MSGPACK", marshaler.NewMsgpackMarshaler())
}

func marshalerBenchmark(name string, marshaler Marshaler) {
	contactID := "64b4dcf8-e0c9-11ee-8c00-dafebafdd11e"
	o := &MarshalerTestObject{
		ID:               258497503316467713,
		State:            "active",
		QwExternalUserID: "wmJawBCQAA7R7dpI98pWhYWzWstkI86A",
		UID:              "6",
		ContactID:        &contactID,
	}

	begin := time.Now()
	for i := 0; i < 10000; i++ {
		_, _ = marshaler.Marshal(o)
	}
	fmt.Printf("%s Marshal: %s\n", name, time.Since(begin))

	s, _ := marshaler.Marshal(o)
	begin = time.Now()
	for i := 0; i < 10000; i++ {
		_ = marshaler.Unmarshal(s, o)
	}
	fmt.Printf("%s Unmarshal: %s\n", name, time.Since(begin))
}
