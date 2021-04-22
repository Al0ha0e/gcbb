package net

import (
	"net"
)

type PSMsgType uint8

const (
	PS_CONN  PSMsgType = iota
	PS_ROUTE PSMsgType = iota
)

type P2PHandlerState uint8

const (
	ST_WAIT P2PHandlerState = iota
	ST_PING P2PHandlerState = iota
	ST_CONN P2PHandlerState = iota
)

type P2PServerMsg struct {
	Type PSMsgType `json:"tp"`
	Data []byte    `json:"data"`
}

type P2PConnMsg struct {
	SrcIP       net.IP `json:"sIP"`
	DstId       int64  `json:"dId"`
	HandlerPort int    `json:"hPort"`
}

type P2PServer interface {
	Init()
	Run()
	ConnectUDP() chan *NetResult
}

type NaiveP2PServer struct {
	Id          int64
	HandlerPool []P2PHandler
	DHT         DistributedHashTable
	HandlerChan chan *NetResult
}

func NewNaiveP2PServer(id int64, pooledCnt int, dht DistributedHashTable) *NaiveP2PServer {
	ret := &NaiveP2PServer{
		Id:          id,
		HandlerPool: make([]P2PHandler, pooledCnt),
		DHT:         dht,
	}
	//Instantiate Handler
	return ret
}

func (nps *NaiveP2PServer) Init() {
	//CONNECT TO STABLE NODES
}

func (nps *NaiveP2PServer) Run() {
	for {
		select {
		case msg := <-nps.HandlerChan:
			nps.DHT.Update(msg.Id)
		}
	}
}

type P2PHandler interface {
	Init(*net.UDPAddr)
	Run()
	SetState(P2PHandlerState)
	GetState() P2PHandlerState
}

type NaiveP2PHandler struct {
	sock     *net.UDPConn
	addr     *net.UDPAddr
	state    P2PHandlerState
	SendChan chan []byte
	RecvChan chan *NetResult
	CtrlChan chan uint8
}

func NewNaiveP2PHandler(sock *net.UDPConn, sendChan chan []byte, recvChan chan *NetResult) *NaiveP2PHandler {
	return &NaiveP2PHandler{
		sock:     sock,
		SendChan: sendChan,
		RecvChan: recvChan,
		CtrlChan: make(chan uint8, 1),
	}
}

func (nph *NaiveP2PHandler) Init(addr *net.UDPAddr) {
	nph.addr = addr
}

func (nph *NaiveP2PHandler) SetState(state P2PHandlerState) {
	nph.state = state
}

func (nph *NaiveP2PHandler) GetState() P2PHandlerState {
	return nph.state
}

func (nph *NaiveP2PHandler) Run() {
	for {
		select {
		case data := <-nph.SendChan:
			Send(nph.sock, data, nph.addr)
		case msg := <-Recv(nph.sock):
			nph.RecvChan <- msg
		case <-nph.CtrlChan:
			return
		}
	}
}
