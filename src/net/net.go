package net

import "net"

func Send(sock *net.UDPConn, data []byte, addr *net.UDPAddr) {
	sock.WriteToUDP(data, addr)
}

func Recv(sock *net.UDPConn) chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		ret := &NetResult{}
		ret.L, ret.Addr, _ = sock.ReadFromUDP(ret.Data)
		result <- ret
	}(ret)
	return ret
}

type NetResult struct {
	Id     uint64
	Status uint8
	Data   []byte
	Addr   *net.UDPAddr
	L      int
}

type NetHandler interface {
	Send(int64, []byte)
	AddListener(string, chan *[]byte)
	RemoveListener(string)
	GetPeers(uint32) []int64
	Run()
}

type NaiveNetHandler struct {
	listeners map[string]chan *[]byte
}

func NewNaiveNetHandler() *NaiveNetHandler {
	ret := &NaiveNetHandler{listeners: make(map[string]chan *[]byte)}
	return ret
}

func (nnh *NaiveNetHandler) Send(peer int64, data []byte) {

}

func (nnh *NaiveNetHandler) AddListener(lid string, listener chan *[]byte) {
	nnh.listeners[lid] = listener
}

func (nnh *NaiveNetHandler) RemoveListener(lid string) {
	delete(nnh.listeners, lid)
}

func Run() {
	for {
	}
}
