package net

import (
	"net"

	"github.com/gcbb/src/common"
)

type NetMsgType uint8

const (
	MSG_PING  NetMsgType = iota
	MSG_PONG  NetMsgType = iota
	MSG_CONN  NetMsgType = iota
	MSG_APPLI NetMsgType = iota
)

type NetMsg struct {
	Src  common.NodeID
	Dst  common.NodeID
	Type NetMsgType
	Data []byte
	//TODO: TTL
}

type NetResult struct {
	Id   common.NodeID
	Data []byte
	Addr *net.UDPAddr
}

type NetHandler interface {
	Send(common.NodeID, []byte)
	AddListener(string, chan *[]byte)
	RemoveListener(string)
	GetPeers(uint32) []common.NodeID
	Run()
}

type NaiveNetHandler struct {
	listeners map[string]chan *[]byte
}

func NewNaiveNetHandler() *NaiveNetHandler {
	ret := &NaiveNetHandler{listeners: make(map[string]chan *[]byte)}
	return ret
}

func (nnh *NaiveNetHandler) Send(peer common.NodeID, data []byte) {

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
