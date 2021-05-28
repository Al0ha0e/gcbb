package net

import (
	"container/list"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gcbb/src/common"
)

var REF_ADDR = &net.UDPAddr{
	IP:   net.IPv4(123, 60, 211, 219),
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
	ConnectUDP(dst common.NodeID)
}

type NaiveP2PServer struct {
	Id            common.NodeID
	HandlerPool   P2PHandlerPool
	DHT           DistributedHashTable
	sendChan      chan *NetMsg
	connectChan   chan common.NodeID
	handlerChan   chan *NetResult
	stateChan     chan *PeerStateChange
	ctrlChan      chan struct{}
	staticHandler P2PHandler
	staticPeers   map[common.NodeID]*net.UDPAddr
	encoder       NetEncoder
}

func NewNaiveP2PServer(
	id common.NodeID,
	pooledCnt int,
	maxPooledCnt int,
	dht DistributedHashTable,
	encoder NetEncoder,
	listenPort int) *NaiveP2PServer {
	hdChan := make(chan *NetResult, 10)
	stChan := make(chan *PeerStateChange, 10)
	sSock, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: listenPort})
	sHandler := NewNaiveP2PHandler(id, sSock, hdChan, stChan, encoder)
	sHandler.Start(true)
	ret := &NaiveP2PServer{
		Id:            id,
		DHT:           dht,
		sendChan:      make(chan *NetMsg, 10),
		connectChan:   make(chan common.NodeID, 10),
		handlerChan:   hdChan,
		stateChan:     stChan,
		ctrlChan:      make(chan struct{}, 1),
		staticHandler: sHandler,
		staticPeers:   make(map[common.NodeID]*net.UDPAddr),
		encoder:       encoder,
	}
	//TODO: INJECT POOL
	ret.HandlerPool = NewNaiveP2PHandlerPool(id, pooledCnt, maxPooledCnt, ret.handlerChan, ret.stateChan, ret.encoder)
	return ret
}

func (nps *NaiveP2PServer) Init(peers []*PeerInfo) {
	//TODO: DEBUG
	for _, peer := range peers {
		nps.staticPeers[peer.Id] = peer.Ip
		nps.ConnectUDP(peer.Id)
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
		fmt.Println("SEND0")
		nps.sendChan <- msg
	}()
}

func (nps *NaiveP2PServer) ConnectUDP(dst common.NodeID) {
	go func() {
		nps.connectChan <- dst
	}()
}

func (nps *NaiveP2PServer) send(msg *NetMsg) {
	fmt.Println("SEND1", msg.Dst)
	addr, has := nps.staticPeers[msg.Dst]
	if has {
		fmt.Println("STATIC SEND")
		nps.staticHandler.Send(nps.encoder.Encode(msg), &PeerInfo{Id: msg.Dst, Ip: addr})
		return
	}

	handler := nps.DHT.Get(msg.Dst)
	state := handler.GetState()
	fmt.Println("STATE WHEN SEND", state)
	if handler != nil && (state == ST_CONN || state == ST_PING) {
		handler.Send(nps.encoder.Encode(msg), nil)
	} else {
		handlers := nps.DHT.GetK(msg.Dst)
		for _, handler = range handlers {
			if handler.GetState() == ST_CONN {
				handler.Send(nps.encoder.Encode(msg), nil)
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

func (nps *NaiveP2PServer) connectUDP(dst common.NodeID, handler P2PHandler) {
	//TODO: MULTIPLE CONNECT MSG
	fmt.Println("TRY SEND CONNECT MSG TO", dst)
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
			fmt.Println("PREPARE CONNECT", id)
			if !nps.DHT.Has(id) {
				handler := nps.HandlerPool.Get()
				//TODO: PEER FAIL BLACKLIST
				handler.Init(nil, id)
				nps.DHT.Insert(handler)
				nps.connectUDP(id, handler)
				handler.Start(false)
			}
		case msg := <-nps.handlerChan:
			nps.DHT.Update(msg.Id)
			var netMsg NetMsg
			nps.encoder.Decode(msg.Data, &netMsg)
			//AUTH
			if netMsg.Dst == nps.Id {
				switch netMsg.Type {
				case MSG_PING:
					fmt.Println("PING!", netMsg.Src)
					//nps.pingPong(netMsg.Src, false)
				//case MSG_PONG:
				case MSG_CONN:
					fmt.Println("ANOTHER TRY CONN", msg.Id)
					var connMsg P2PConnMsg
					nps.encoder.Decode(netMsg.Data, &connMsg)
					handler := nps.DHT.Get(netMsg.Src)
					if handler == nil { //COUNTERPART POSITIVE
						handler = nps.HandlerPool.Get()
						nps.DHT.Insert(handler)
						nps.connectUDP(netMsg.Src, handler)
						handler.Start(false)
					}
					if handler.GetState() == ST_WAIT {
						handler.Init(&net.UDPAddr{
							IP:   connMsg.IP,
							Port: connMsg.Port,
						}, netMsg.Src)
						handler.Connect()
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
			dstId := state.Id
			handler := state.Handler
			if state.State == ST_CONN {
				fmt.Println("CONNECT!", nps.Id, dstId)
			} else if state.State == ST_WAIT {
				fmt.Println("RETRY CONNECT")
				nps.connectUDP(dstId, handler)
			} else if state.State == ST_PING {
				fmt.Println("SEND PING")
				nps.pingPong(dstId, true)
			} else if state.State == ST_DISCONN {
				fmt.Println("GG DISCONN")
				handler.Stop()
				nps.DHT.Remove(dstId)
				nps.HandlerPool.Set(handler)
			}
		case <-nps.ctrlChan:
			fmt.Println("STOP")
			nps.staticHandler.Stop()
			return
		}
	}
}

type P2PHandler interface {
	Init(addr *net.UDPAddr, id common.NodeID)
	Start(isStatic bool)
	Connect()
	Send(data []byte, peer *PeerInfo)
	Stop()
	SetState(state P2PHandlerState)
	GetState() P2PHandlerState
	GetDstId() common.NodeID
	GenConnMsg() *P2PConnMsg
	Dispose()
}

type NaiveP2PHandler struct {
	srcId     common.NodeID
	dstId     common.NodeID
	sock      *net.UDPConn
	dstAddr   *net.UDPAddr
	pubIp     net.IP
	pubPort   int
	state     P2PHandlerState
	stateLock sync.RWMutex
	sendChan  chan *struct {
		data []byte
		peer *PeerInfo
	}
	recvChan     chan *NetResult
	stateChan    chan *PeerStateChange
	ctrlChan     chan struct{}
	connectChan  chan struct{}
	pingTicker   *time.Ticker
	waitTicker   *time.Ticker
	disconnTimer *time.Timer
	encoder      NetEncoder
}

func NewNaiveP2PHandler(id common.NodeID, sock *net.UDPConn, recvChan chan *NetResult, stateChan chan *PeerStateChange, encoder NetEncoder) *NaiveP2PHandler {
	ret := &NaiveP2PHandler{
		srcId: id,
		sock:  sock,
		sendChan: make(chan *struct {
			data []byte
			peer *PeerInfo
		}, 10),
		recvChan:    recvChan,
		stateChan:   stateChan,
		ctrlChan:    make(chan struct{}, 2),
		connectChan: make(chan struct{}, 2),
		encoder:     encoder,
	}
	ret.getSelfPubIp()
	return ret
}

func (nph *NaiveP2PHandler) Init(addr *net.UDPAddr, id common.NodeID) {
	nph.dstAddr = addr
	nph.dstId = id
}

func (nph *NaiveP2PHandler) Start(isStatic bool) {
	nph.disconnTimer = time.NewTimer(100 * time.Second)
	nph.waitTicker = time.NewTicker(10 * time.Second)
	nph.pingTicker = time.NewTicker(10 * time.Second)
	if isStatic {
		nph.SetState(ST_CONN)
		nph.disconnTimer.Stop()
		nph.waitTicker.Stop()
	} else {
		nph.SetState(ST_WAIT)
	}
	nph.pingTicker.Stop()
	go nph.run(isStatic)
}

func (nph *NaiveP2PHandler) Connect() {
	go func() {
		nph.connectChan <- struct{}{}
	}()
}

func (nph *NaiveP2PHandler) Send(data []byte, peer *PeerInfo) {
	go func() {
		nph.sendChan <- &struct {
			data []byte
			peer *PeerInfo
		}{data, peer}
	}()
}

func (nph *NaiveP2PHandler) Stop() {
	go func() {
		nph.ctrlChan <- struct{}{}
	}()
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

func (nph *NaiveP2PHandler) GetDstId() common.NodeID {
	return nph.dstId
}

func (nph *NaiveP2PHandler) GenConnMsg() *P2PConnMsg {
	return &P2PConnMsg{
		IP:   nph.pubIp,
		Port: nph.pubPort,
	}
}

func (nph *NaiveP2PHandler) Dispose() {
	nph.sock.Close()
}

//CAUTION !!! BLOCK
func (nph *NaiveP2PHandler) getSelfPubIp() {
	ticker := time.NewTicker(1 * time.Second)
	nph.sock.WriteToUDP([]byte(""), REF_ADDR)
	data := make([]byte, 256)
	infochan := make(chan int, 1)
	go func(c chan int) {
		l, _, _ := nph.sock.ReadFromUDP(data)
		infochan <- l
	}(infochan)
	for {
		select {
		case <-ticker.C:
			ticker.Reset(1 * time.Second)
			nph.sock.WriteToUDP([]byte(""), REF_ADDR)
		case l := <-infochan:
			data = data[:l]
			ipPort := strings.Split(string(data), ":")
			fmt.Println("IP PORT ", ipPort)
			nph.pubIp = net.ParseIP(ipPort[0])
			nph.pubPort, _ = strconv.Atoi(ipPort[1])
			return
		}
	}
}

func (nph *NaiveP2PHandler) send(data []byte, peer *PeerInfo) {
	msg := P2PMsg{
		SrcId:  nph.srcId,
		DstId:  nph.dstId,
		Data:   data,
		Digest: make([]byte, 0),
		Sign:   make([]byte, 0),
	}
	if peer != nil {
		msg.DstId = peer.Id
		nph.sock.WriteToUDP(nph.encoder.Encode(&msg), peer.Ip)
		fmt.Println("static send to", peer.Id, peer.Ip)
	} else {
		nph.sock.WriteToUDP(nph.encoder.Encode(&msg), nph.dstAddr)
		fmt.Println("send to", nph.dstId, nph.dstAddr)
	}

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
			fmt.Println("RECV!! FROM", msg.SrcId, "TO", msg.DstId)
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

//TODO
//TIMER HARDCODE
func (nph *NaiveP2PHandler) run(isStatic bool) {
	if isStatic {
		for {
			select {
			case msg := <-nph.recv():
				go func() {
					nph.recvChan <- msg
				}()
			case info := <-nph.sendChan:
				nph.send(info.data, info.peer)
			case <-nph.ctrlChan:
				return
			}
		}
	}
	for {
		select {
		case info := <-nph.sendChan:
			nph.pingTicker.Reset(10 * time.Second)
			nph.send(info.data, nil)
		case <-nph.connectChan:
			if nph.GetState() == ST_WAIT {
				nph.waitTicker.Stop()
				nph.pingTicker.Reset(10 * time.Second)
				nph.disconnTimer.Reset(100 * time.Second)
				nph.SetState(ST_PING)
			}
		case msg := <-nph.recv():
			state := nph.GetState()
			if state == ST_DISCONN {
				continue
			}
			if state != ST_CONN {
				if state == ST_WAIT {
					nph.Init(msg.Addr, msg.Id)
					nph.waitTicker.Stop()
					nph.pingTicker = time.NewTicker(10 * time.Second)
				}
				nph.SetState(ST_CONN)
				go func() {
					nph.stateChan <- &PeerStateChange{
						Id:      nph.dstId,
						State:   nph.state,
						Handler: nph,
					}
				}()
			}
			nph.disconnTimer.Reset(100 * time.Second)
			go func() {
				nph.recvChan <- msg
			}()
		case _, ok := <-nph.waitTicker.C:
			if ok {
				go func() {
					nph.stateChan <- &PeerStateChange{
						Id:      nph.dstId,
						State:   ST_WAIT,
						Handler: nph,
					}
				}()
			}
		case _, ok := <-nph.pingTicker.C:
			if ok {
				//nph.SetState(ST_PING)
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
				nph.waitTicker.Stop()
				nph.pingTicker.Stop()
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
			nph.waitTicker.Stop()
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
