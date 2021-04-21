package net

import (
	"net"
	"time"
)

type P2PServerMsg struct {
	Type uint8  `json:"tp"`
	Data []byte `json:"data"`
}

type P2PConnMsg struct {
	Type uint8 `json:"tp"`

	DstIp          net.IP
	SrcServerPort  int
	SrcHandlerPort int
	DstServerPort  int
	DstHandlerPort int
}

type P2PServer interface {
	SetServer(*net.UDPAddr) chan *NetResult
	ConnectUDP()
	Run()
}

type NaiveP2PServer struct {
}

type P2PHandler interface {
	SetServer(*net.UDPAddr) chan *NetResult
	//ConnectTCP()
	ConnectUDP()
	Send([]byte, *net.UDPAddr)
	Recv() chan *NetResult
}

type NaiveP2PHandler struct {
	sock *net.UDPConn
}

func NewNaiveP2PHandler(sock *net.UDPConn) *NaiveP2PHandler {
	return &NaiveP2PHandler{sock: sock}
}

func (nph *NaiveP2PHandler) SetServer(addr *net.UDPAddr) chan *NetResult {
	ret := make(chan *NetResult, 1)
	go nph.setServer(addr, ret)
	return ret
}

func (nph *NaiveP2PHandler) setServer(addr *net.UDPAddr, result chan *NetResult) {
	data := []byte("conn")
	nph.Send(data, addr)
	severConnTimer := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-severConnTimer.C:
			ret := &NetResult{Status: 1}
			result <- ret
		case <-nph.Recv():
			ret := &NetResult{Status: 0}
			result <- ret
		}
	}
}

// func (nph *NaiveP2PHandler) ConnectTCP() {}

func (nph *NaiveP2PHandler) ConnectUDP() {}

func (nph *NaiveP2PHandler) Send(data []byte, addr *net.UDPAddr) {
	nph.sock.WriteToUDP(data, addr)
}

func (nph *NaiveP2PHandler) Recv() chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		ret := &NetResult{}
		ret.L, ret.Addr, _ = nph.sock.ReadFromUDP(ret.Data)
		result <- ret
	}(ret)
	return ret
}
