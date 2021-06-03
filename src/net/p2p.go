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
	// IP: net.IPv4(123, 60, 211, 219),
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

type P2PMsgType uint8

const (
	PM_PING P2PMsgType = iota
	PM_NET  P2PMsgType = iota
)

type P2PMsg struct {
	SrcId  common.NodeID
	DstId  common.NodeID
	Type   P2PMsgType
	Data   []byte
	Digest []byte
	Sign   []byte
}

type P2PConnMsg struct {
	SrcIP   net.IP
	SrcPort int
}

type PeerInfo struct {
	PeerId common.NodeID
	PeerIp *net.UDPAddr
}

type PeerStateChange struct {
	PeerId  common.NodeID
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
	SelfId        common.NodeID
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
	selfId common.NodeID,
	pooledCnt int,
	maxPooledCnt int,
	dht DistributedHashTable,
	encoder NetEncoder,
	listenPort int) *NaiveP2PServer {
	hdChan := make(chan *NetResult, 10)
	stChan := make(chan *PeerStateChange, 10)
	sSock, _ := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(0, 0, 0, 0), Port: listenPort})
	sHandler := NewNaiveP2PHandler(selfId, sSock, encoder)
	sHandler.SetChannel(hdChan, stChan)
	sHandler.Start(true)
	ret := &NaiveP2PServer{
		SelfId:        selfId,
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
	ret.HandlerPool = NewNaiveP2PHandlerPool(selfId, pooledCnt, maxPooledCnt, ret.encoder)
	return ret
}

func (nps *NaiveP2PServer) Init(peers []*PeerInfo) {
	for _, peer := range peers {
		nps.staticPeers[peer.PeerId] = peer.PeerIp
		nps.ConnectUDP(peer.PeerId)
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
	fmt.Println("SEND1", msg.DstId)
	addr, has := nps.staticPeers[msg.DstId]

	handler := nps.DHT.Get(msg.DstId)
	fmt.Println("STATE WHEN SEND", handler.GetState())
	if (handler != nil) && handler.ReadyToSend() {
		handler.Send(nps.encoder.Encode(msg), nil)
	} else if has {
		fmt.Println("STATIC SEND")
		nps.staticHandler.Send(nps.encoder.Encode(msg), &PeerInfo{PeerId: msg.DstId, PeerIp: addr})
	} else {
		handlers := nps.DHT.GetK(msg.DstId)
		for _, handler = range handlers {
			if handler.GetState() == ST_CONN {
				handler.Send(nps.encoder.Encode(msg), nil)
			}
		}
	}
}

func (nps *NaiveP2PServer) connectUDP(dst common.NodeID, handler P2PHandler) {
	//TODO: MULTIPLE CONNECT MSG
	fmt.Println("TRY SEND CONNECT MSG TO", dst)
	connMsg := handler.GenConnMsg()
	msg := &NetMsg{
		SrcId: nps.SelfId,
		DstId: dst,
		Type:  MSG_CONN,
		Data:  nps.encoder.Encode(connMsg),
		TTL:   MAX_TTL,
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
				handler.SetChannel(nps.handlerChan, nps.stateChan)
				//TODO: PEER FAIL BLACKLIST
				handler.Init(nil, id)
				nps.DHT.Insert(handler)
				nps.connectUDP(id, handler)
				handler.Start(false)
			}
		case msg := <-nps.handlerChan:
			nps.DHT.Update(msg.SrcId)
			var netMsg NetMsg
			nps.encoder.Decode(msg.Data, &netMsg)
			//AUTH
			fmt.Println("SERVER RECV FROM", netMsg.SrcId, "TO", netMsg.DstId)
			if netMsg.DstId == nps.SelfId {
				switch netMsg.Type {
				case MSG_CONN:
					fmt.Println("ANOTHER TRY CONN", netMsg.SrcId)
					var connMsg P2PConnMsg
					nps.encoder.Decode(netMsg.Data, &connMsg)
					handler := nps.DHT.Get(netMsg.SrcId)
					if handler == nil { //COUNTERPART POSITIVE
						handler = nps.HandlerPool.Get()
						handler.SetChannel(nps.handlerChan, nps.stateChan)
						handler.Init(&net.UDPAddr{
							IP:   connMsg.SrcIP,
							Port: connMsg.SrcPort,
						}, netMsg.SrcId)
						nps.DHT.Insert(handler)
						nps.connectUDP(netMsg.SrcId, handler)
						handler.Start(false)
						handler.Connect()
					} else if handler.GetState() == ST_WAIT {
						handler.Init(&net.UDPAddr{
							IP:   connMsg.SrcIP,
							Port: connMsg.SrcPort,
						}, netMsg.SrcId)
						handler.Connect()
					}
				case MSG_SINGLE:
					continue
				case MSG_MULTI:
					continue
				}
			} else {
				fmt.Println("ROUTE TO", netMsg.DstId, netMsg.TTL)
				if netMsg.TTL > 0 {
					netMsg.TTL -= 1
					nps.Send(&netMsg)
				}
			}

		case state := <-nps.stateChan:
			dstId := state.PeerId
			handler := state.Handler
			if state.State == ST_CONN {
				fmt.Println("CONNECT!", nps.SelfId, dstId)
				//TODO Init Multi Handler
			} else if state.State == ST_WAIT {
				fmt.Println("POSITIVE RETRY CONNECT")
				nps.connectUDP(dstId, handler)
			} else if state.State == ST_PING {
				fmt.Println("PASSIVE RETRY CONNECT")
				nps.connectUDP(dstId, handler)
			} else if state.State == ST_DISCONN {
				fmt.Println("GG DISCONN")
				handler.Stop()
				nps.DHT.Remove(dstId)
				nps.HandlerPool.Set(handler)
				//TODO Dispose Multi Handler
			}
		case <-nps.ctrlChan:
			fmt.Println("STOP")
			nps.staticHandler.Stop()
			return
		}
	}
}

type P2PHandler interface {
	SetChannel(recvChan chan *NetResult, stateChan chan *PeerStateChange)
	Init(dstAddr *net.UDPAddr, dstId common.NodeID)
	Start(isStatic bool)
	Connect()
	ReadyToSend() bool
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
	waitTicker2  *time.Ticker
	disconnTimer *time.Timer
	encoder      NetEncoder
}

func NewNaiveP2PHandler(id common.NodeID, sock *net.UDPConn, encoder NetEncoder) *NaiveP2PHandler {
	ret := &NaiveP2PHandler{
		srcId: id,
		sock:  sock,
		sendChan: make(chan *struct {
			data []byte
			peer *PeerInfo
		}, 10),
		ctrlChan:    make(chan struct{}, 2),
		connectChan: make(chan struct{}, 2),
		encoder:     encoder,
	}
	ret.getSelfPubIp()
	return ret
}

func (nph *NaiveP2PHandler) SetChannel(recvChan chan *NetResult, stateChan chan *PeerStateChange) {
	nph.recvChan = recvChan
	nph.stateChan = stateChan
}

func (nph *NaiveP2PHandler) Init(dstAddr *net.UDPAddr, dstId common.NodeID) {
	nph.dstAddr = dstAddr
	nph.dstId = dstId
}

func (nph *NaiveP2PHandler) Start(isStatic bool) {
	nph.disconnTimer = time.NewTimer(100 * time.Second)
	nph.waitTicker = time.NewTicker(10 * time.Second)
	nph.waitTicker2 = time.NewTicker(20 * time.Second)
	nph.pingTicker = time.NewTicker(10 * time.Second)
	if isStatic {
		nph.SetState(ST_CONN)
		nph.disconnTimer.Stop()
		nph.waitTicker.Stop()
	} else {
		nph.SetState(ST_WAIT)
	}
	nph.waitTicker2.Stop()
	nph.pingTicker.Stop()
	go nph.run(isStatic)
}

func (nph *NaiveP2PHandler) Connect() {
	go func() {
		nph.connectChan <- struct{}{}
	}()
}

func (nph *NaiveP2PHandler) ReadyToSend() bool {
	if nph.GetState() != ST_DISCONN && nph.dstAddr != nil {
		return true
	}
	return false
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
		SrcIP:   nph.pubIp,
		SrcPort: nph.pubPort,
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

func (nph *NaiveP2PHandler) send(data []byte, tp P2PMsgType, peer *PeerInfo) {
	msg := P2PMsg{
		SrcId:  nph.srcId,
		DstId:  nph.dstId,
		Type:   tp,
		Data:   data,
		Digest: make([]byte, 0),
		Sign:   make([]byte, 0),
	}
	if peer != nil {
		msg.DstId = peer.PeerId
		nph.sock.WriteToUDP(nph.encoder.Encode(&msg), peer.PeerIp)
		fmt.Println("static send to", peer.PeerId, peer.PeerIp)
	} else {
		pdata := nph.encoder.Encode(&msg)
		nph.sock.WriteToUDP(pdata, nph.dstAddr)
		fmt.Println("send to", nph.dstId, nph.dstAddr, len(pdata))
	}

}

func (nph *NaiveP2PHandler) recv() chan *NetResult {
	ret := make(chan *NetResult, 1)
	go func(result chan *NetResult) {
		for {
			data := make([]byte, 1024)
			ret := &NetResult{}
			var l int
			var msg P2PMsg
			var err error
			l, ret.SrcAddr, err = nph.sock.ReadFromUDP(data)
			if err != nil {
				fmt.Println(err)
			}
			data = data[:l]
			nph.encoder.Decode(data, &msg)
			//AUTH
			//fmt.Println("RECV!! FROM", msg.SrcId, "TO", msg.DstId, l)
			if msg.DstId == nph.srcId {
				state := nph.GetState()
				//fmt.Println("STATE WHEN RECV", state)
				if state == ST_DISCONN {
					return
				}

				nph.disconnTimer.Reset(100 * time.Second)

				if msg.Type == PM_PING {
					//fmt.Println("PING RECV FROM", msg.SrcId, ret.SrcAddr)
					if state == ST_PING {
						nph.waitTicker2.Stop()
						nph.SetState(ST_CONN)
						go func() {
							nph.stateChan <- &PeerStateChange{
								PeerId:  nph.dstId,
								State:   nph.state,
								Handler: nph,
							}
						}()
					}
					result <- nil
					return
				} else {
					fmt.Println("NORMAL RECV FROM", msg.SrcId, ret.SrcAddr)
					ret.SrcId = msg.SrcId
					ret.Data = msg.Data
					result <- ret
					return
				}
			}
		}
	}(ret)
	return ret
}

func (nph *NaiveP2PHandler) ping() {
	nph.send([]byte{}, PM_PING, nil)
}

//TODO
//TIMER HARDCODE
func (nph *NaiveP2PHandler) run(isStatic bool) {
	msgChan := nph.recv()
	if isStatic {
		for {
			select {
			case msg := <-msgChan:
				msgChan = nph.recv()
				if msg == nil {
					continue
				}
				go func() {
					fmt.Println("STATIC HANDLER RECV", msg.SrcId)
					nph.recvChan <- msg
				}()
			case info := <-nph.sendChan:
				nph.send(info.data, PM_NET, info.peer)
			case <-nph.ctrlChan:
				return
			}
		}
	}
	for {
		select {
		case info := <-nph.sendChan:
			nph.pingTicker.Reset(10 * time.Second)
			nph.send(info.data, PM_NET, nil)
		case <-nph.connectChan:
			if nph.GetState() == ST_WAIT {
				nph.waitTicker.Stop()
				nph.waitTicker2.Reset(20 * time.Second)
				nph.pingTicker.Reset(10 * time.Second)
				nph.disconnTimer.Reset(100 * time.Second)
				nph.SetState(ST_PING)
			}
		case msg := <-msgChan:
			msgChan = nph.recv()
			if msg == nil {
				continue
			}
			go func() {
				fmt.Println("HANDLER RECV", msg.SrcId)
				nph.recvChan <- msg
			}()
		case _, ok := <-nph.waitTicker.C:
			if ok {
				go func() {
					nph.stateChan <- &PeerStateChange{
						PeerId:  nph.dstId,
						State:   ST_WAIT,
						Handler: nph,
					}
				}()
			}
		case _, ok := <-nph.waitTicker2.C:
			if ok {
				go func() {
					nph.stateChan <- &PeerStateChange{
						PeerId:  nph.dstId,
						State:   ST_PING,
						Handler: nph,
					}
				}()
			}
		case _, ok := <-nph.pingTicker.C:
			if ok {
				nph.ping()
			}
		case _, ok := <-nph.disconnTimer.C:
			if ok {
				nph.SetState(ST_DISCONN)
				nph.waitTicker.Stop()
				nph.waitTicker2.Stop()
				nph.pingTicker.Stop()
				go func() {
					nph.stateChan <- &PeerStateChange{
						PeerId:  nph.dstId,
						State:   nph.state,
						Handler: nph,
					}
				}()
			}
		case <-nph.ctrlChan:
			nph.pingTicker.Stop()
			nph.waitTicker.Stop()
			nph.waitTicker2.Stop()
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
	pool         *list.List
	encoder      NetEncoder
}

func NewNaiveP2PHandlerPool(id common.NodeID, pooledCnt int, maxPooledCnt int, encoder NetEncoder) *NaiveP2PHandlerPool {
	ret := &NaiveP2PHandlerPool{
		id:           id,
		pooledCnt:    pooledCnt,
		maxPooledCnt: maxPooledCnt,
		encoder:      encoder,
	}
	ret.pool = list.New().Init()
	for i := 0; i < pooledCnt; i++ {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		ret.pool.PushFront(NewNaiveP2PHandler(id, sock, encoder))
	}
	return ret
}

func (nphp *NaiveP2PHandlerPool) Get() P2PHandler {
	if nphp.pool.Front() == nil {
		sock, _ := net.ListenUDP("udp4", &net.UDPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 0,
		})
		return NewNaiveP2PHandler(nphp.id, sock, nphp.encoder)
	}
	ret := nphp.pool.Front().Value.(P2PHandler)
	nphp.pool.Remove(nphp.pool.Front())
	return ret
}

func (nphp *NaiveP2PHandlerPool) Set(handler P2PHandler) {
	if nphp.pool.Len() == nphp.maxPooledCnt {
		handler.Dispose()
		return
	}
	nphp.pool.PushFront(handler)
}
