package net

import (
	"container/list"
	"math"
	"time"

	"github.com/gcbb/src/common"
)

type MultipackState struct {
	StateMap     uint8
	MapStSeq     uint32
	ExpectSeq    uint32
	MaxExpectCnt uint32
}

type PackInfo struct {
	PackID     uint32
	PackSeq    uint32
	SubPackCnt uint32
	Seq        uint32
}

type MultipackMsg struct {
	State MultipackState
	Info  PackInfo
	Data  []byte
}

const MP_STATE_SIZE = 1 + 4*3
const MP_INFO_SIZE = 4 * 4
const MULTIPACK_DATA_SIZE = NETMSG_DATA_SIZE - MP_STATE_SIZE - MP_INFO_SIZE - 4

type LoopedQueue struct {
	data    []interface{}
	queueSt uint32
	queueEn uint32
	size    uint32
	stSeq   uint32
	enSeq   uint32
}

func NewLoopedQueue(size uint32, seq uint32) *LoopedQueue {
	return &LoopedQueue{
		data:    make([]interface{}, size+1),
		queueSt: 0,
		queueEn: 0,
		stSeq:   seq,
		enSeq:   seq,
		size:    size,
	}
}

func (lq *LoopedQueue) Full() bool {
	return (lq.queueEn-lq.queueSt+lq.size+1)%(lq.size+1) == lq.size
}

func (lq *LoopedQueue) Length() uint32 {
	return (lq.queueEn - lq.queueSt + lq.size + 1) % (lq.size + 1)
}

func (lq *LoopedQueue) IsSendingSeq(seq uint32) bool {
	return seq-lq.stSeq < lq.Length()
}

func (lq *LoopedQueue) Get(id uint32) interface{} {
	if (id-lq.queueSt+lq.size+1)%(lq.size+1) < lq.Length() {
		return lq.data[id]
	}
	return nil
}

func (lq *LoopedQueue) GetBySeq(seq uint32) interface{} {
	if lq.IsSendingSeq(seq) {
		return lq.Get(seq - lq.stSeq)
	}
	return nil
}

func (lq *LoopedQueue) GetEnSeq() uint32 {
	return lq.enSeq
}

func (lq *LoopedQueue) PopFront() interface{} {
	if lq.queueSt == lq.queueEn {
		return nil
	}
	ret := lq.data[lq.queueSt]
	lq.queueSt = (lq.queueSt + 1) % (lq.size + 1)
	lq.stSeq += 1
	return ret
}

func (lq *LoopedQueue) PopToSeq(seq uint32) {
	if lq.IsSendingSeq(seq) {
		for ; lq.stSeq != seq; lq.PopFront() {
		}
	}
}

func (lq *LoopedQueue) PushBack(obj interface{}) bool {
	if (lq.queueEn-lq.queueSt+lq.size+1)%(lq.size+1) != lq.size {
		lq.data[lq.queueEn] = obj
		lq.queueEn = (lq.queueEn + 1) % (lq.size + 1)
		lq.enSeq += 1
		return true
	}
	return false
}

type MultiPackHandler interface {
	Init()
	Start()
	Send(data []byte)
	OnPackArrive(data []byte)
	Stop()
}

type NaiveMultiPackHandler struct {
	server            P2PServer
	srcID             common.NodeID
	dstID             common.NodeID
	encoder           common.Encoder
	ctrlChan          chan struct{}
	sendChan          chan *MultipackMsg
	recvChan          chan []byte
	sendingQueue      *LoopedQueue
	receiveQueue      *LoopedQueue
	waitingList       *list.List
	partialDataList   *list.List
	maxWaitingListLen uint32
	statusTimer       *time.Timer
	packID            uint32
}

func NewNaiveMultiPackHandler(server P2PServer, srcID common.NodeID, dstID common.NodeID, encoder common.Encoder) *NaiveMultiPackHandler {
	//TODO
	return &NaiveMultiPackHandler{
		server:   server,
		srcID:    srcID,
		dstID:    dstID,
		encoder:  encoder,
		ctrlChan: make(chan struct{}, 1),
		sendChan: make(chan *MultipackMsg, 10),
		recvChan: make(chan []byte, 10),
	}
}

func (nmph *NaiveMultiPackHandler) Init() {
	nmph.statusTimer = time.NewTimer(10 * time.Second)
	nmph.statusTimer.Stop()
}

func (nmph *NaiveMultiPackHandler) Start() {
	nmph.statusTimer.Reset(10 * time.Second)
	go nmph.run()
}

func (nmph *NaiveMultiPackHandler) Send(data []byte) { //BLOCKED
	packNum := len(data) / MULTIPACK_DATA_SIZE
	remBytes := len(data) % MULTIPACK_DATA_SIZE
	subpackCnt := uint32(math.Ceil(float64(len(data)) / float64(MULTIPACK_DATA_SIZE)))
	nmph.packID = (nmph.packID + 1) % math.MaxUint32

	for i := 0; i < packNum; i++ {
		info := PackInfo{
			PackID:     nmph.packID,
			PackSeq:    uint32(i),
			SubPackCnt: subpackCnt,
		}
		msg := &MultipackMsg{
			Info: info,
			Data: data[i*MULTIPACK_DATA_SIZE : (i+1)*MULTIPACK_DATA_SIZE],
		}
		nmph.sendChan <- msg
	}

	if remBytes > 0 {
		info := PackInfo{
			PackID:     nmph.packID,
			PackSeq:    uint32(packNum),
			SubPackCnt: subpackCnt,
		}
		msg := &MultipackMsg{
			Info: info,
			Data: data[packNum*MULTIPACK_DATA_SIZE:],
		}
		nmph.sendChan <- msg
	}
}

func (nmph *NaiveMultiPackHandler) OnPackArrive(data []byte) {
	nmph.recvChan <- data
}

func (nmph *NaiveMultiPackHandler) Stop() {
	nmph.ctrlChan <- struct{}{}
}

func (nmph *NaiveMultiPackHandler) getState() MultipackState {
	//TODO
	return MultipackState{}
}

func (nmph *NaiveMultiPackHandler) syncState(state MultipackState) {
	stMap := state.StateMap
	stSeq := state.MapStSeq
	expSeq := state.ExpectSeq
	expCnt := state.MaxExpectCnt
	//remove successfully transmitted packs
	nmph.sendingQueue.PopToSeq(stSeq)
	//Retransmit In Map
	var i uint32
	for i = stSeq; i != stSeq+8 && i != expSeq; i++ {
		if !nmph.sendingQueue.IsSendingSeq(i) {
			break
		}
		if (stMap & (1 << (i - stSeq))) > 0 {
			msg := nmph.sendingQueue.GetBySeq(i).(*MultipackMsg)
			nmph.send(msg)
			//TODO Estimate TTL
		}
	}

	for i = 0; i < expCnt; i++ {
		seq := expSeq + i
		if nmph.sendingQueue.IsSendingSeq(seq) {
			//Retransmit
			msg := nmph.sendingQueue.GetBySeq(seq).(*MultipackMsg)
			nmph.send(msg)
		} else {
			if seq != nmph.sendingQueue.GetEnSeq()+1 {
				//invalid seq
				break
			}
			if nmph.sendingQueue.Full() {
				//TODO: Congestion Control
				break
			}
			if nmph.waitingList.Len() == 0 {
				if len(nmph.sendChan) == 0 {
					break
				}
				msg := <-nmph.sendChan
				nmph.waitingList.PushBack(msg)
			}
			msg := nmph.waitingList.Front().Value.(*MultipackMsg)
			nmph.sendingQueue.PushBack(msg)
			//TODO Set Timer for msg
			nmph.send(msg)
		}
	}

	//Add data into waitingList
	for nmph.waitingList.Len() < int(nmph.maxWaitingListLen) && len(nmph.sendChan) > 0 {
		nmph.waitingList.PushBack(<-nmph.sendChan)
	}
}

func (nmph *NaiveMultiPackHandler) handlePackReceive(msg *MultipackMsg) {
	//TODO

}

func (nmph *NaiveMultiPackHandler) send(msg *MultipackMsg) {
	nmsg := &NetMsg{
		SrcId: nmph.srcID,
		DstId: nmph.dstID,
		Type:  MSG_MULTI,
		TTL:   MAX_TTL,
	}
	if msg == nil {
		//send state
		nmsg.Data = nmph.encoder.Encode(nmph.getState())
	} else {
		nmsg.Data = nmph.encoder.Encode(msg)
	}
	nmph.statusTimer.Reset(5 * time.Second)
	nmph.server.Send(nmsg)
}

func (nmph *NaiveMultiPackHandler) run() {
	for {
		select {
		case <-nmph.statusTimer.C:
			nmph.send(nil)
			nmph.statusTimer.Reset(5 * time.Second)
		case data := <-nmph.recvChan:
			if len(data) == MP_STATE_SIZE { //State Only
				var state MultipackState
				nmph.encoder.Decode(data, &state)
				nmph.syncState(state)
			} else { //With Data
				var msg MultipackMsg
				nmph.encoder.Decode(data, &msg)
				nmph.syncState(msg.State)
				nmph.handlePackReceive(&msg)
			}
		case <-nmph.ctrlChan:
			nmph.statusTimer.Stop()
		}
	}
}
