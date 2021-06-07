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

const NETMSG_DATA_SIZE = NETMSG_PACK_SIZE - 40 - 1 - 4 - 4 //TODO

type NetResult struct {
	SrcId   common.NodeID
	Data    []byte
	SrcAddr *net.UDPAddr
}

type NetHandler interface {
	Send(common.NodeID, []byte)
	Init(peers []*PeerInfo)
	Start()
	ConnectP2P(dst common.NodeID)
	// AddListener(string, chan *[]byte)
	// RemoveListener(string)
	// GetPeers(uint32) []common.NodeID
}

type NaiveNetHandler struct {
	server          P2PServer
	multiHandlerMap map[common.NodeID]MultiPackHandler
	// listeners map[string]chan *[]byte
}

func NewNaiveNetHandler(server P2PServer) *NaiveNetHandler {
	// ret := &NaiveNetHandler{listeners: make(map[string]chan *[]byte)}
	ret := &NaiveNetHandler{
		server: server,
	}
	return ret
}

func (nnh *NaiveNetHandler) Send(peer common.NodeID, data []byte) {

}

func (nnh *NaiveNetHandler) Init(peers []*PeerInfo) {
	nnh.server.Init(peers)
}

func (nnh *NaiveNetHandler) Start() {
	nnh.server.Start()
	go nnh.run()
}

func (nnh *NaiveNetHandler) ConnectP2P(dst common.NodeID) {
	nnh.server.ConnectUDP(dst)
}

// func (nnh *NaiveNetHandler) AddListener(lid string, listener chan *[]byte) {
// 	nnh.listeners[lid] = listener
// }

// func (nnh *NaiveNetHandler) RemoveListener(lid string) {
// 	delete(nnh.listeners, lid)
// }

func (nnh *NaiveNetHandler) run() {

}
