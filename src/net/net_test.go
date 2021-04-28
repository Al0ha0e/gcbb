package net

import (
	"net"
	"testing"
)

func TestEncode(t *testing.T) {
	var coder GobNetEncoder
	val1 := P2PMsg{
		SrcId: [20]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	}
	t.Log(val1.SrcId)
	data := coder.Encode(&val1)
	val2 := P2PMsg{}
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
