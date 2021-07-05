package net

import (
	"net"
	"time"

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

func (nnh *NaiveNetHandler) run() {

}

var EmuChanMap map[common.NodeID]chan *NetMsg

type NetHandlerEmulator struct {
	NodeID     common.NodeID
	handlerMap map[uint16]AppliNetHandler
	encoder    common.Encoder
	recvChan   chan *NetMsg
	ctrlChan   chan struct{}
}

func NewNetHandlerEmulator(nodeId common.NodeID) *NetHandlerEmulator {
	ret := &NetHandlerEmulator{
		NodeID:     nodeId,
		handlerMap: make(map[uint16]AppliNetHandler),
		recvChan:   make(chan *NetMsg, 10),
		ctrlChan:   make(chan struct{}, 1),
	}
	EmuChanMap[ret.NodeID] = ret.recvChan
	return ret
}

func (emulator *NetHandlerEmulator) AddAppliHandler(handler AppliNetHandler) {
	emulator.handlerMap[handler.GetID()] = handler
}

func (emulator *NetHandlerEmulator) SendTo(peer common.NodeID, msg *AppliNetMsg) {
	go func() {
		timer := time.NewTimer(200 * time.Millisecond)
		<-timer.C
		EmuChanMap[peer] <- &NetMsg{
			SrcId: emulator.NodeID,
			DstId: peer,
			Data:  emulator.encoder.Encode(msg),
		}
	}()
}

func (emulator *NetHandlerEmulator) ReliableSendTo(peer common.NodeID, msg *AppliNetMsg, id uint32, resultChan chan *SendResult) {
	go func() {
		timer := time.NewTimer(200 * time.Millisecond)
		<-timer.C
		EmuChanMap[peer] <- &NetMsg{
			SrcId: emulator.NodeID,
			DstId: peer,
			Data:  emulator.encoder.Encode(msg),
		}
		resultChan <- &SendResult{
			ID:            id,
			OK:            true,
			DstPeerID:     peer,
			DstHandlerID:  msg.DstHandlerID,
			DstListenerID: msg.DstListenerID,
		}
	}()
}

func (emulator *NetHandlerEmulator) Start() {
	go emulator.run()
}

func (emulator *NetHandlerEmulator) run() {
	for {
		select {
		case msg := <-emulator.recvChan:
			var appliMsg AppliNetMsg
			emulator.encoder.Decode(msg.Data, &appliMsg)
			emulator.handlerMap[appliMsg.DstHandlerID].OnMsgArrive(msg.SrcId, &appliMsg)
		case <-emulator.ctrlChan:
			delete(EmuChanMap, emulator.NodeID)
			return
		}
	}
}
