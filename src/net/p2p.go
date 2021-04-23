package net

import (
	"container/list"
	"net"
	"time"
)

type PSMsgType uint8

const (
	PS_CONN  PSMsgType = iota
	PS_ROUTE PSMsgType = iota
)

type P2PHandlerState uint8

const (
	ST_WAIT    P2PHandlerState = iota
	ST_PING    P2PHandlerState = iota
	ST_CONN    P2PHandlerState = iota
	ST_DISCONN P2PHandlerState = iota
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

type PeerInfo struct {
	Id uint64
	Ip net.UDPAddr
}

type PeerStateChange struct {
	Id      uint64
	State   P2PHandlerState
	Handler P2PHandler
}

type P2PServer interface {
	Init()
	Start()
	Stop()
	ConnectUDP() chan *NetResult
}

type NaiveP2PServer struct {
	Id          int64
	HandlerPool *list.List
	DHT         DistributedHashTable
	HandlerChan chan *NetResult
	StateChan   chan *PeerStateChange
	ctrlChan    chan struct{}
}

func NewNaiveP2PServer(id int64, pooledCnt int, dht DistributedHashTable) *NaiveP2PServer {
	ret := &NaiveP2PServer{
		Id:          id,
		HandlerPool: list.New().Init(),
		DHT:         dht,
		HandlerChan: make(chan *NetResult, 10),
		StateChan:   make(chan *PeerStateChange, 10),
	}
	for i := 0; i < pooledCnt; i++ {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		ret.HandlerPool.PushFront(NewNaiveP2PHandler(sock, ret.HandlerChan, ret.StateChan))
	}
	return ret
}

func (nps *NaiveP2PServer) Init(peers []PeerInfo) {

}

func (nps *NaiveP2PServer) Start() {
	go nps.run()
}

func (nps *NaiveP2PServer) Stop() {
	nps.ctrlChan <- struct{}{}
}

func (nps *NaiveP2PServer) run() {
	for {
		select {
		case msg := <-nps.HandlerChan:
			nps.DHT.Update(msg.Id)
		case state := <-nps.StateChan:
			handler := state.Handler
			if state.State == ST_PING {
				handler.Send([]byte("ping"))
			} else if state.State == ST_DISCONN {
				handler.Stop()
				nps.DHT.Remove(state.Id)
				nps.HandlerPool.PushFront(handler)
			}
		case <-nps.ctrlChan:
			return
		}
	}
}

type P2PHandler interface {
	Init(*net.UDPAddr, uint64)
	Start()
	Send([]byte)
	Stop()
	SetState(P2PHandlerState)
	GetState() P2PHandlerState
	GetId() uint64
}

type NaiveP2PHandler struct {
	id           uint64
	sock         *net.UDPConn
	addr         *net.UDPAddr
	state        P2PHandlerState
	sendChan     chan []byte
	recvChan     chan *NetResult
	stateChan    chan *PeerStateChange
	ctrlChan     chan struct{}
	conn2Ping    *time.Timer
	ping2Disconn *time.Timer
}

func NewNaiveP2PHandler(sock *net.UDPConn, recvChan chan *NetResult, stateChan chan *PeerStateChange) *NaiveP2PHandler {
	return &NaiveP2PHandler{
		sock:      sock,
		sendChan:  make(chan []byte, 1),
		recvChan:  recvChan,
		stateChan: stateChan,
		ctrlChan:  make(chan struct{}, 1),
	}
}

func (nph *NaiveP2PHandler) Init(addr *net.UDPAddr, id uint64) {
	nph.addr = addr
	nph.id = id
}

func (nph *NaiveP2PHandler) Send(data []byte) {
	nph.sendChan <- data
}

func (nph *NaiveP2PHandler) Stop() {
	nph.ctrlChan <- struct{}{}
}

func (nph *NaiveP2PHandler) SetState(state P2PHandlerState) {
	nph.state = state
}

func (nph *NaiveP2PHandler) GetState() P2PHandlerState {
	return nph.state
}

func (nph *NaiveP2PHandler) GetId() uint64 {
	return nph.id
}

func (nph *NaiveP2PHandler) Start() {
	nph.conn2Ping = time.NewTimer(1 * time.Second)
	nph.ping2Disconn = time.NewTimer(100 * time.Second)
	nph.ping2Disconn.Stop()
	go nph.run()
}

func (nph *NaiveP2PHandler) run() {
	for {
		select {
		case data := <-nph.sendChan:
			Send(nph.sock, data, nph.addr)
		case msg := <-Recv(nph.sock):
			nph.recvChan <- msg
			nph.conn2Ping.Reset(100 * time.Second)
			nph.ping2Disconn.Stop()
		case _, ok := <-nph.conn2Ping.C:
			if ok {
				nph.state = ST_PING
				nph.stateChan <- &PeerStateChange{
					Id:      nph.id,
					State:   nph.state,
					Handler: nph,
				}
				nph.ping2Disconn = time.NewTimer(100 * time.Second)
			}
		case _, ok := <-nph.ping2Disconn.C:
			if ok {
				nph.state = ST_DISCONN
				nph.stateChan <- &PeerStateChange{
					Id:      nph.id,
					State:   nph.state,
					Handler: nph,
				}
			}
		case <-nph.ctrlChan:
			nph.conn2Ping.Stop()
			nph.ping2Disconn.Stop()
			return
		}
	}
}
