package net

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"net"
	"time"

	"github.com/gcbb/src/common"
)

type P2PHandlerState uint8

const (
	ST_WAIT    P2PHandlerState = iota
	ST_PING    P2PHandlerState = iota
	ST_CONN    P2PHandlerState = iota
	ST_DISCONN P2PHandlerState = iota
)

type P2PMsg struct {
	Id     common.NodeID
	Data   []byte
	Digest []byte
	Sign   []byte
}

type P2PConnMsg struct {
	Address string
}

type PeerInfo struct {
	Id common.NodeID
	Ip *net.UDPAddr
}

type PeerStateChange struct {
	Id      common.NodeID
	State   P2PHandlerState
	Handler P2PHandler
}

type P2PServer interface {
	Init()
	Start()
	Stop()
	Send(NetMsg)
	ConnectUDP() chan *NetResult
}

type NaiveP2PServer struct {
	Id          common.NodeID
	HandlerPool P2PHandlerPool
	DHT         DistributedHashTable
	sendChan    chan *NetMsg
	HandlerChan chan *NetResult
	StateChan   chan *PeerStateChange
	ctrlChan    chan struct{}
}

func NewNaiveP2PServer(id common.NodeID, pooledCnt int, maxPooledCnt int, dht DistributedHashTable) *NaiveP2PServer {
	ret := &NaiveP2PServer{
		Id:          id,
		DHT:         dht,
		sendChan:    make(chan *NetMsg, 10),
		HandlerChan: make(chan *NetResult, 10),
		StateChan:   make(chan *PeerStateChange, 10),
	}
	ret.HandlerPool = NewNaiveP2PHandlerPool(id, pooledCnt, maxPooledCnt, ret.HandlerChan, ret.StateChan)
	return ret
}

func (nps *NaiveP2PServer) Init(peers []PeerInfo) {
	for _, peer := range peers {
		handler := nps.HandlerPool.Get()
		handler.Init(peer.Ip, peer.Id)
		nps.DHT.Insert(handler)
		handler.Start()
	}
}

func (nps *NaiveP2PServer) Start() {
	go nps.run()
}

func (nps *NaiveP2PServer) Stop() {
	nps.ctrlChan <- struct{}{}
}

func (nps *NaiveP2PServer) Send(msg *NetMsg) {
	nps.sendChan <- msg
}

func (nps *NaiveP2PServer) pingPong(dst common.NodeID, isPing bool) {
	msg := &NetMsg{
		Src:  nps.Id,
		Dst:  dst,
		Data: make([]byte, 0),
	}
	if isPing {
		msg.Type = MSG_PING
	} else {
		msg.Type = MSG_PONG
	}
	nps.Send(msg)
}

func (nps *NaiveP2PServer) send(msg *NetMsg) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(msg)
	node := nps.DHT.Get(msg.Dst)
	if node != nil {
		node.Send(buf.Bytes())
	} else {
		nodes := nps.DHT.GetK()
		for _, node = range nodes {
			node.Send(buf.Bytes())
		}
	}
}

func (nps *NaiveP2PServer) connectUDP(handler P2PHandler) {

}

func (nps *NaiveP2PServer) run() {
	for {
		select {
		case msg := <-nps.sendChan:
			nps.send(msg)
		case msg := <-nps.HandlerChan:
			nps.DHT.Update(msg.Id)
			var netMsg NetMsg
			decoder := gob.NewDecoder(bytes.NewReader(msg.Data))
			decoder.Decode(&netMsg)
			//AUTH
			if netMsg.Dst == nps.Id {
				switch netMsg.Type {
				case MSG_PING:
					nps.pingPong(netMsg.Src, false)
				case MSG_PONG:
				case MSG_CONN:
					decoder := gob.NewDecoder(bytes.NewReader(netMsg.Data))
					var connMsg P2PConnMsg
					decoder.Decode(&connMsg)
					handler := nps.DHT.Get(netMsg.Src)
					if handler == nil { //COUNTERPART POSITIVE
						handler = nps.HandlerPool.Get()
						handler.Init(&net.UDPAddr{IP: connMsg.SrcIP, Port: connMsg.HandlerPort}, netMsg.Src)
						nps.DHT.Insert(handler)

					}
					handler.SetState(ST_PING)
					handler.Start()
				case MSG_APPLI:
				}
			} else {
				nps.Send(&netMsg)
			}

		case state := <-nps.StateChan:
			handler := state.Handler
			if state.State == ST_PING {
				nps.pingPong(state.Id, true)
			} else if state.State == ST_DISCONN {
				handler.Stop()
				nps.DHT.Remove(state.Id)
				nps.HandlerPool.Set(handler)
			}
		case <-nps.ctrlChan:
			return
		}
	}
}

type P2PHandler interface {
	Init(*net.UDPAddr, common.NodeID)
	Start()
	Send([]byte)
	Stop()
	SetState(P2PHandlerState)
	GetState() P2PHandlerState
	GetSelfAddr() net.Addr
	GetDstId() common.NodeID
	Dispose()
}

type NaiveP2PHandler struct {
	srcId        common.NodeID
	dstId        common.NodeID
	sock         *net.UDPConn
	dstAddr      *net.UDPAddr
	state        P2PHandlerState
	sendChan     chan []byte
	recvChan     chan *NetResult
	stateChan    chan *PeerStateChange
	ctrlChan     chan struct{}
	conn2Ping    *time.Timer
	ping2Disconn *time.Timer
}

func NewNaiveP2PHandler(id common.NodeID, sock *net.UDPConn, recvChan chan *NetResult, stateChan chan *PeerStateChange) *NaiveP2PHandler {
	return &NaiveP2PHandler{
		srcId:     id,
		sock:      sock,
		sendChan:  make(chan []byte, 1),
		recvChan:  recvChan,
		stateChan: stateChan,
		ctrlChan:  make(chan struct{}, 1),
	}
}

func (nph *NaiveP2PHandler) Init(addr *net.UDPAddr, id common.NodeID) {
	nph.dstAddr = addr
	nph.dstId = id
}

func (nph *NaiveP2PHandler) send(data []byte) {
	msg := P2PMsg{
		Id:     nph.srcId,
		Data:   data,
		Digest: make([]byte, 0),
		Sign:   make([]byte, 0),
	}
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(msg)
	nph.sock.WriteToUDP(buf.Bytes(), nph.dstAddr)
}

func (nph *NaiveP2PHandler) recv() chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		ret := &NetResult{}
		data := make([]byte, 1024)
		var l int
		l, ret.Addr, _ = nph.sock.ReadFromUDP(data)
		data = data[:l]
		decoder := gob.NewDecoder(bytes.NewReader(data))
		var msg P2PMsg
		decoder.Decode(&msg)
		//AUTH
		ret.Id = msg.Id
		ret.Data = msg.Data
		result <- ret
	}(ret)
	return ret
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

func (nph *NaiveP2PHandler) GetSelfAddr() net.Addr {
	return nph.sock.LocalAddr().String()
}

func (nph *NaiveP2PHandler) GetDstId() common.NodeID {
	return nph.dstId
}

func (nph *NaiveP2PHandler) Start() {
	nph.conn2Ping = time.NewTimer(1 * time.Second)
	nph.ping2Disconn = time.NewTimer(100 * time.Second)
	nph.ping2Disconn.Stop()
	go nph.run()
}

func (nph *NaiveP2PHandler) Dispose() {
	nph.sock.Close()
}

func (nph *NaiveP2PHandler) run() {
	for {
		select {
		case data := <-nph.sendChan:
			nph.send(data)
		case msg := <-nph.recv():
			nph.recvChan <- msg
			nph.conn2Ping.Reset(100 * time.Second)
			nph.ping2Disconn.Stop()
		case _, ok := <-nph.conn2Ping.C:
			if ok {
				nph.state = ST_PING
				nph.stateChan <- &PeerStateChange{
					Id:      nph.dstId,
					State:   nph.state,
					Handler: nph,
				}
				nph.ping2Disconn = time.NewTimer(100 * time.Second)
			}
		case _, ok := <-nph.ping2Disconn.C:
			if ok {
				nph.state = ST_DISCONN
				nph.stateChan <- &PeerStateChange{
					Id:      nph.dstId,
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

type P2PHandlerPool interface {
	Get() P2PHandler
	Set(P2PHandler)
}

type NaiveP2PHandlerPool struct {
	id           common.NodeID
	pooledCnt    int
	maxPooledCnt int
	handlerChan  chan *NetResult
	stateChan    chan *PeerStateChange
	pool         *list.List
}

func NewNaiveP2PHandlerPool(id common.NodeID, pooledCnt int, maxPooledCnt int, handlerChan chan *NetResult, stateChan chan *PeerStateChange) *NaiveP2PHandlerPool {
	ret := &NaiveP2PHandlerPool{
		id:           id,
		pooledCnt:    pooledCnt,
		maxPooledCnt: maxPooledCnt,
		handlerChan:  handlerChan,
		stateChan:    stateChan,
	}
	ret.pool = list.New().Init()
	for i := 0; i < pooledCnt; i++ {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		ret.pool.PushFront(NewNaiveP2PHandler(id, sock, handlerChan, stateChan))
	}
	return ret
}

func (nphp *NaiveP2PHandlerPool) Get() P2PHandler {
	if nphp.pool.Front() == nil {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		return NewNaiveP2PHandler(nphp.id, sock, nphp.handlerChan, nphp.stateChan)
	}
	return nphp.pool.Front().Value.(P2PHandler)
}

func (nphp *NaiveP2PHandlerPool) Set(handler P2PHandler) {
	if nphp.pool.Len() == nphp.maxPooledCnt {
		handler.Dispose()
		return
	}
	nphp.pool.PushFront(handler)
}
