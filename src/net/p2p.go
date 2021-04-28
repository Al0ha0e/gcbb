package net

import (
	"bytes"
	"container/list"
	"encoding/gob"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gcbb/src/common"
)

var REF_ADDR = &net.UDPAddr{
	IP:   net.IPv4(127, 0, 0, 1),
	Port: 2233,
}

type P2PHandlerState uint8

const (
	ST_WAIT    P2PHandlerState = iota
	ST_PING    P2PHandlerState = iota
	ST_CONN    P2PHandlerState = iota
	ST_DISCONN P2PHandlerState = iota
)

type P2PMsg struct {
	SrcId  common.NodeID
	DstId  common.NodeID
	Data   []byte
	Digest []byte
	Sign   []byte
}

type P2PConnMsg struct {
	IP   net.IP
	Port int
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
	ConnectUDP(common.NodeID)
}

type NaiveP2PServer struct {
	Id          common.NodeID
	HandlerPool P2PHandlerPool
	DHT         DistributedHashTable
	sendChan    chan *NetMsg
	connectChan chan common.NodeID
	handlerChan chan *NetResult
	stateChan   chan *PeerStateChange
	ctrlChan    chan struct{}
	encoder     NetEncoder
}

func NewNaiveP2PServer(id common.NodeID, pooledCnt int, maxPooledCnt int, dht DistributedHashTable, encoder NetEncoder) *NaiveP2PServer {
	ret := &NaiveP2PServer{
		Id:          id,
		DHT:         dht,
		sendChan:    make(chan *NetMsg, 10),
		handlerChan: make(chan *NetResult, 10),
		stateChan:   make(chan *PeerStateChange, 10),
		encoder:     encoder,
	}
	//TODO: INJECT POOL
	ret.HandlerPool = NewNaiveP2PHandlerPool(id, pooledCnt, maxPooledCnt, ret.handlerChan, ret.stateChan, ret.encoder)
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
	go func() {
		nps.sendChan <- msg
	}()
}

func (nps *NaiveP2PServer) send(msg *NetMsg) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(msg)
	handler := nps.DHT.Get(msg.Dst)
	if handler != nil && handler.GetState() == ST_CONN {
		handler.Send(buf.Bytes())
	} else {
		handlers := nps.DHT.GetK(msg.Dst)
		for _, handler = range handlers {
			if handler.GetState() == ST_CONN {
				handler.Send(buf.Bytes())
			}
		}
	}
}

func (nps *NaiveP2PServer) pingPong(dst common.NodeID, isPing bool) {
	msg := &NetMsg{
		Src:  nps.Id,
		Dst:  dst,
		Data: make([]byte, 0),
		TTL:  MAX_TTL,
	}
	msg.Type = MSG_PING
	//if isPing {
	//	msg.Type = MSG_PING
	//} else {
	//	msg.Type = MSG_PONG
	//}
	nps.Send(msg)
}

func (nps *NaiveP2PServer) ConnectUDP(dst common.NodeID) {
	go func() {
		nps.connectChan <- dst
	}()
}

func (nps *NaiveP2PServer) connectUDP(dst common.NodeID, handler P2PHandler) {
	connMsg := handler.GenConnMsg()
	msg := &NetMsg{
		Src:  nps.Id,
		Dst:  dst,
		Type: MSG_CONN,
		Data: nps.encoder.Encode(connMsg),
		TTL:  MAX_TTL,
	}
	nps.Send(msg)
}

func (nps *NaiveP2PServer) run() {
	for {
		select {
		case msg := <-nps.sendChan:
			nps.send(msg)
		case id := <-nps.connectChan:
			if !nps.DHT.Has(id) {
				handler := nps.HandlerPool.Get()
				handler.SetState(ST_WAIT)
				nps.DHT.Insert(handler)
				nps.connectUDP(id, handler)
			}
		case msg := <-nps.handlerChan:
			nps.DHT.Update(msg.Id)
			var netMsg NetMsg
			nps.encoder.Decode(msg.Data, &netMsg)
			//AUTH
			if netMsg.Dst == nps.Id {
				switch netMsg.Type {
				case MSG_PING:
					//nps.pingPong(netMsg.Src, false)
				//case MSG_PONG:
				case MSG_CONN:
					var connMsg P2PConnMsg
					nps.encoder.Decode(netMsg.Data, &connMsg)
					handler := nps.DHT.Get(netMsg.Src)
					if handler == nil { //COUNTERPART POSITIVE
						handler = nps.HandlerPool.Get()
						handler.SetState(ST_WAIT)
						nps.DHT.Insert(handler)
						nps.connectUDP(netMsg.Src, handler)
					}
					if handler.GetState() == ST_WAIT {
						handler.Init(&net.UDPAddr{
							IP:   connMsg.IP,
							Port: connMsg.Port,
						}, netMsg.Src)
						handler.SetState(ST_PING)
						handler.Start()
					}
				case MSG_APPLI:
				}
			} else {
				if netMsg.TTL > 0 {
					netMsg.TTL -= 1
					nps.Send(&netMsg)
				}
			}

		case state := <-nps.stateChan:
			handler := state.Handler
			if state.State == ST_CONN {
			} else if state.State == ST_PING {
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
	GetDstId() common.NodeID
	GenConnMsg() *P2PConnMsg
	Dispose()
}

type NaiveP2PHandler struct {
	srcId        common.NodeID
	dstId        common.NodeID
	sock         *net.UDPConn
	dstAddr      *net.UDPAddr
	pubIp        net.IP
	pubPort      int
	state        P2PHandlerState
	stateLock    sync.RWMutex
	sendChan     chan []byte
	recvChan     chan *NetResult
	stateChan    chan *PeerStateChange
	ctrlChan     chan struct{}
	pingTicker   *time.Ticker
	disconnTimer *time.Timer
	encoder      NetEncoder
}

func NewNaiveP2PHandler(id common.NodeID, sock *net.UDPConn, recvChan chan *NetResult, stateChan chan *PeerStateChange, encoder NetEncoder) *NaiveP2PHandler {

	ret := &NaiveP2PHandler{
		srcId:     id,
		sock:      sock,
		sendChan:  make(chan []byte, 10),
		recvChan:  recvChan,
		stateChan: stateChan,
		ctrlChan:  make(chan struct{}, 2),
		encoder:   encoder,
	}
	ret.getSelfPubIp()
	return ret
}

//CAUTION !!! BLOCK
func (nph *NaiveP2PHandler) getSelfPubIp() {
	nph.sock.WriteToUDP([]byte(""), REF_ADDR)
	data := make([]byte, 256)
	l, _, _ := nph.sock.ReadFromUDP(data)
	data = data[:l]
	ipPort := strings.Split(string(data), ":")
	nph.pubIp = net.ParseIP(ipPort[0])
	nph.pubPort, _ = strconv.Atoi(ipPort[1])
}

func (nph *NaiveP2PHandler) Init(addr *net.UDPAddr, id common.NodeID) {
	nph.dstAddr = addr
	nph.dstId = id
}

func (nph *NaiveP2PHandler) send(data []byte) {
	msg := P2PMsg{
		SrcId:  nph.srcId,
		DstId:  nph.dstId,
		Data:   data,
		Digest: make([]byte, 0),
		Sign:   make([]byte, 0),
	}
	nph.sock.WriteToUDP(nph.encoder.Encode(&msg), nph.dstAddr)
}

func (nph *NaiveP2PHandler) recv() chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		for {
			ret := &NetResult{}
			data := make([]byte, 1024)
			var l int
			l, ret.Addr, _ = nph.sock.ReadFromUDP(data)
			data = data[:l]
			var msg P2PMsg
			nph.encoder.Decode(data, &msg)
			//AUTH
			if msg.DstId == nph.srcId {
				ret.Id = msg.SrcId
				ret.Data = msg.Data
				result <- ret
				return
			}
		}
	}(ret)
	return ret
}

func (nph *NaiveP2PHandler) Send(data []byte) {
	go func() {
		nph.sendChan <- data
	}()
}

func (nph *NaiveP2PHandler) Stop() {
	nph.ctrlChan <- struct{}{}
}

func (nph *NaiveP2PHandler) SetState(state P2PHandlerState) {
	nph.stateLock.Lock()
	nph.state = state
	nph.stateLock.Unlock()
}

func (nph *NaiveP2PHandler) GetState() P2PHandlerState {
	nph.stateLock.RLock()
	ret := nph.state
	nph.stateLock.RUnlock()
	return ret
}

func (nph *NaiveP2PHandler) GenConnMsg() *P2PConnMsg {
	return &P2PConnMsg{
		IP:   nph.pubIp,
		Port: nph.pubPort,
	}
}

func (nph *NaiveP2PHandler) GetDstId() common.NodeID {
	return nph.dstId
}

func (nph *NaiveP2PHandler) Start() {
	nph.pingTicker = time.NewTicker(10 * time.Second)
	nph.disconnTimer = time.NewTimer(100 * time.Second)
	nph.disconnTimer.Stop()
	go nph.run()
}

func (nph *NaiveP2PHandler) Dispose() {
	nph.sock.Close()
}

//TODO
//TIMER HARDCODE
func (nph *NaiveP2PHandler) run() {
	for {
		select {
		case data := <-nph.sendChan:
			nph.send(data)
		case msg := <-nph.recv():
			if nph.GetState() != ST_CONN {
				nph.SetState(ST_CONN)
				go func() {
					nph.stateChan <- &PeerStateChange{
						Id:      nph.dstId,
						State:   nph.state,
						Handler: nph,
					}
				}()
			}
			nph.pingTicker.Reset(10 * time.Second)
			nph.disconnTimer.Reset(100 * time.Second)
			go func() {
				nph.recvChan <- msg
			}()
		case _, ok := <-nph.pingTicker.C:
			if ok {
				//nph.SetState(ST_PING)
				nph.pingTicker.Reset(10 * time.Second)
				go func() {
					nph.stateChan <- &PeerStateChange{
						Id:      nph.dstId,
						State:   ST_PING,
						Handler: nph,
					}
				}()
				//nph.ping2Disconn = time.NewTimer(100 * time.Second)
			}
		case _, ok := <-nph.disconnTimer.C:
			if ok {
				nph.SetState(ST_DISCONN)
				go func() {
					nph.stateChan <- &PeerStateChange{
						Id:      nph.dstId,
						State:   nph.state,
						Handler: nph,
					}
				}()
			}
		case <-nph.ctrlChan:
			nph.pingTicker.Stop()
			nph.disconnTimer.Stop()
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
	encoder      NetEncoder
}

func NewNaiveP2PHandlerPool(id common.NodeID, pooledCnt int, maxPooledCnt int, handlerChan chan *NetResult, stateChan chan *PeerStateChange, encoder NetEncoder) *NaiveP2PHandlerPool {
	ret := &NaiveP2PHandlerPool{
		id:           id,
		pooledCnt:    pooledCnt,
		maxPooledCnt: maxPooledCnt,
		handlerChan:  handlerChan,
		stateChan:    stateChan,
		encoder:      encoder,
	}
	ret.pool = list.New().Init()
	for i := 0; i < pooledCnt; i++ {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		ret.pool.PushFront(NewNaiveP2PHandler(id, sock, handlerChan, stateChan, encoder))
	}
	return ret
}

func (nphp *NaiveP2PHandlerPool) Get() P2PHandler {
	if nphp.pool.Front() == nil {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		return NewNaiveP2PHandler(nphp.id, sock, nphp.handlerChan, nphp.stateChan, nphp.encoder)
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
