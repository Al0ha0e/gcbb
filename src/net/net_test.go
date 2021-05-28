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

func TestNewHandler(t *testing.T) {
	sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0,
	})
	hdc := make(chan *NetResult, 1)
	stc := make(chan *PeerStateChange, 1)
	NewNaiveP2PHandler([20]byte{0}, sock, hdc, stc, &GobNetEncoder{})
}

func TestNewServer(t *testing.T) {
	id := [20]byte{0}
	dht := NewKadDHT(id, 5)
	NewNaiveP2PServer(id, 2, 3, dht, &GobNetEncoder{}, 8090)
}
