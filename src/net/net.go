package net

import (
	"net"

	"github.com/gcbb/src/common"
)

type NetMsgType uint8

const MAX_TTL = 65

const (
	MSG_CONN   NetMsgType = iota
	MSG_SINGLE NetMsgType = iota
	MSG_MULTI  NetMsgType = iota
)

type NetMsg struct {
	SrcId common.NodeID `nvencoder:"nodeid"`
	DstId common.NodeID `nvencoder:"nodeid"`
	Type  NetMsgType
	Data  []byte
	TTL   uint32
}

type NetResult struct {
	SrcId   common.NodeID
	Data    []byte
	SrcAddr *net.UDPAddr
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
	// for {
	// }
}
