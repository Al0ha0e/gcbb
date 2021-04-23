package net

import "net"

type NetMsgType uint8

const (
	PS_PING  NetMsgType = iota
	PS_CONN  NetMsgType = iota
	PS_APPLI NetMsgType = iota
)

type NetMsg struct {
	Src    int64      `json:"src"`
	Dst    int64      `json:"dst"`
	Type   NetMsgType `json:"tp"`
	Data   []byte     `json:"data"`
	Digest []byte     `json:"digest"`
	Sign   []byte     `json:"sign"`
}

func Send(sock *net.UDPConn, data []byte, addr *net.UDPAddr) {
	sock.WriteToUDP(data, addr)
}

func Recv(sock *net.UDPConn) chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		ret := &NetResult{
			Data: make([]byte, 1024),
		}
		var l int
		l, ret.Addr, _ = sock.ReadFromUDP(ret.Data)
		ret.Data = ret.Data[:l]
		result <- ret
	}(ret)
	return ret
}

type NetResult struct {
	Data []byte
	Addr *net.UDPAddr
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
