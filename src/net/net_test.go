package net

import (
	"net"
	"testing"

	"github.com/gcbb/src/common"
)

func TestNewHandler(t *testing.T) {
	sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: 0,
	})
	NewNaiveP2PHandler([20]byte{0}, sock, &common.GobNetEncoder{})
}

func TestNewServer(t *testing.T) {
	id := [20]byte{0}
	dht := NewKadDHT(id, 5)
	NewNaiveP2PServer(id, 2, 3, dht, &common.GobNetEncoder{}, 8090)
}
