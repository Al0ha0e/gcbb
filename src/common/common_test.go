package common

import (
	"fmt"
	"net"
	"testing"
)

type TestStruct struct {
	SrcId NodeID `nvencoder:"nodeid"`
	DstId NodeID `nvencoder:"nodeid"`
	Data  []byte
}

func TestEncode(t *testing.T) {
	var coder GobNetEncoder
	val1 := TestStruct{
		SrcId: [20]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}
	t.Log(val1.SrcId)
	data := coder.Encode(&val1)
	val2 := TestStruct{}
	coder.Decode(data, &val2)
	t.Log(val2.SrcId)

	val3 := net.UDPAddr{
		IP:   net.IPv4(1, 2, 3, 4),
		Port: 8080,
	}
	t.Log(val3)
	data = coder.Encode(&val3)
	var val4 net.UDPAddr
	coder.Decode(data, &val4)
	t.Log(val4)
}

func TestNaiveEncoder(t *testing.T) {
	val1 := TestStruct{
		SrcId: [20]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		DstId: [20]byte{3},
		Data:  []byte{4, 5, 6},
	}
	var coder NaiveNetEncoder
	data := coder.Encode(val1)
	fmt.Println(len(data), data)
	val2 := TestStruct{}
	coder.Decode(data, &val2)
	fmt.Println(val2)
}
