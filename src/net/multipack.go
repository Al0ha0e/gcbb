package net

import (
	"container/list"
	"context"
	"math"
	"time"

	"github.com/gcbb/src/common"
)

const (
	TimerAlpha float32 = 0.125
	TimerBeta  float32 = 0.25
	TimerMu    float32 = 1
	TimerDet   float32 = 4
)

type MultiPackHandlerState uint8

const (
	ST_SLOW_START MultiPackHandlerState = iota
	ST_CONG_AVOID MultiPackHandlerState = iota
)

const (
	EVENT_ACK     = iota
	EVENT_NAK     = iota
	EVENT_TIMEOUT = iota
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
	SendTime   int64
}

type MultipackMsg struct {
	State MultipackState
	Info  PackInfo
	Data  []byte
}

const MP_STATE_SIZE = 1 + 4*3
const MP_INFO_SIZE = 4 * 4
const MULTIPACK_DATA_SIZE = NETMSG_DATA_SIZE - MP_STATE_SIZE - MP_INFO_SIZE - 4

type MultiPackHandler interface {
	Init()
	Start()
	Send(data []byte)
	OnPackArrive(data []byte)
	GetAssembledChan() chan []byte
	Stop()
}

type NaiveMultiPackHandler struct {
	server             P2PServer
	srcID              common.NodeID
	dstID              common.NodeID
	encoder            common.Encoder
	ctrlChan           chan struct{}
	retansmitChan      chan uint32
	sendChan           chan *MultipackMsg
	recvChan           chan []byte
	assembledChan      chan []byte
	rootContext        context.Context
	rootCancelFunc     context.CancelFunc
	sendingQueue       *LoopedQueue
	receiveQueue       *LoopedQueue
	waitingList        *list.List
	partialDataList    *list.List
	maxWaitingListLen  float32
	slowStartThreshold float32
	statusTimer        *time.Timer
	retransmitTimerMap map[uint32]*common.Timer
	packID             uint32
	rto                float32
	srtt               float32
	devrtt             float32
	state              MultiPackHandlerState
}

func NewNaiveMultiPackHandler(server P2PServer, srcID common.NodeID, dstID common.NodeID, encoder common.Encoder) *NaiveMultiPackHandler {
	//TODO
	ctx, fnc := context.WithCancel(context.Background())
	return &NaiveMultiPackHandler{
		server:         server,
		srcID:          srcID,
		dstID:          dstID,
		encoder:        encoder,
		ctrlChan:       make(chan struct{}, 1),
		retansmitChan:  make(chan uint32, 10),
		sendChan:       make(chan *MultipackMsg, 10),
		recvChan:       make(chan []byte, 10),
		assembledChan:  make(chan []byte, 4),
		rootContext:    ctx,
		rootCancelFunc: fnc,
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
	nmph.packID = nmph.packID + 1

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

func (nmph *NaiveMultiPackHandler) GetAssembledChan() chan []byte {
	return nmph.assembledChan
}

func (nmph *NaiveMultiPackHandler) Stop() {
	nmph.ctrlChan <- struct{}{}
}

func (nmph *NaiveMultiPackHandler) getState() MultipackState {
	receiveQueue := nmph.receiveQueue
	stateMap := uint8(0)
	st := receiveQueue.GetStSeq()
	for i := st; i-st != 8 && i != receiveQueue.GetEnSeq(); i++ {
		if receiveQueue.GetBySeq(i) != nil {
			stateMap |= 1 << (i - st)
		}
	}
	ret := MultipackState{
		StateMap:     stateMap,
		MapStSeq:     receiveQueue.GetStSeq(),
		ExpectSeq:    receiveQueue.GetEnSeq(),
		MaxExpectCnt: receiveQueue.Size() - receiveQueue.Length(),
	}
	return ret
}

func (nmph *NaiveMultiPackHandler) calcRTT(rtt float32) {
	if nmph.srtt == 0 {
		nmph.srtt = rtt
		nmph.devrtt = rtt / 2
	} else {
		nmph.srtt = nmph.srtt + TimerAlpha*(rtt-nmph.srtt)
		nmph.devrtt = (1-TimerBeta)*nmph.devrtt + TimerBeta*float32(math.Abs(float64(rtt)-float64(nmph.srtt)))
	}

	nmph.rto = nmph.srtt*TimerMu + nmph.devrtt*TimerDet
}

func (nmph *NaiveMultiPackHandler) calcCwnd(cond uint8) {
	if cond == EVENT_ACK { //ack
		if nmph.state == ST_SLOW_START {
			nmph.maxWaitingListLen += 1
			if nmph.maxWaitingListLen >= nmph.slowStartThreshold {
				nmph.state = ST_CONG_AVOID
			}
		} else if nmph.state == ST_CONG_AVOID {
			nmph.maxWaitingListLen += 1 / nmph.maxWaitingListLen
		}
	} else if cond == EVENT_NAK { //nak
		nmph.maxWaitingListLen /= 2
		nmph.slowStartThreshold = nmph.maxWaitingListLen
		nmph.state = ST_CONG_AVOID
	} else if cond == EVENT_TIMEOUT { //timeout
		nmph.slowStartThreshold = nmph.maxWaitingListLen / 2
		nmph.maxWaitingListLen = 1
		nmph.state = ST_SLOW_START
	}
}

func (nmph *NaiveMultiPackHandler) syncState(state MultipackState) {
	stMap := state.StateMap
	stSeq := state.MapStSeq
	expSeq := state.ExpectSeq
	expCnt := state.MaxExpectCnt
	sq := nmph.sendingQueue
	//remove successfully transmitted packs
	if sq.IsValidSeq(stSeq) {
		for ; sq.GetStSeq() != stSeq; sq.PopFront() {
			seq := sq.GetStSeq()
			if _, ok := nmph.retransmitTimerMap[seq]; ok {
				nmph.retransmitTimerMap[seq].Stop()
				delete(nmph.retransmitTimerMap, seq)
				nmph.calcRTT(float32(time.Now().Unix()) - float32(sq.GetBySeq(seq).(*MultipackMsg).Info.SendTime))
				nmph.calcCwnd(EVENT_ACK)
			}
		}
	}
	//Retransmit In Map
	var i uint32
	for i = stSeq; i != stSeq+8 && i != expSeq; i++ {
		if !sq.IsValidSeq(i) {
			break
		}
		if (stMap & (1 << (i - stSeq))) == 0 {
			msg := sq.GetBySeq(i).(*MultipackMsg)
			nmph.calcCwnd(EVENT_NAK)
			nmph.send(msg)
		} else {
			if _, ok := nmph.retransmitTimerMap[i]; ok {
				nmph.retransmitTimerMap[i].Stop()
				delete(nmph.retransmitTimerMap, i)
				nmph.calcRTT(float32(time.Now().Unix()) - float32(sq.GetBySeq(i).(*MultipackMsg).Info.SendTime))
				nmph.calcCwnd(EVENT_ACK)
			}
		}
	}

	for i = 0; i < expCnt; i++ {
		seq := expSeq + i
		if sq.IsValidSeq(seq) {
			//Retransmit
			msg := sq.GetBySeq(seq).(*MultipackMsg)
			sendTime := msg.Info.SendTime
			if 2*(time.Now().Unix()-sendTime) >= int64(nmph.srtt) {
				//TODO refine
				nmph.send(msg)
			}
		} else {
			if seq != sq.GetEnSeq() {
				//invalid seq
				break
			}
			if sq.Full() {
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
			nmph.waitingList.Remove(nmph.waitingList.Front())
			msg.Info.Seq = seq
			sq.PushBack(msg)
			nmph.send(msg)
		}
	}

	//Add data into waitingList
	for nmph.waitingList.Len() < int(nmph.maxWaitingListLen) && len(nmph.sendChan) > 0 {
		nmph.waitingList.PushBack(<-nmph.sendChan)
	}
}

func (nmph *NaiveMultiPackHandler) handlePackReceive(msg *MultipackMsg) {
	seq := msg.Info.Seq
	recvQueue := nmph.receiveQueue
	if recvQueue.IsValidSeq(seq) {
		recvQueue.SetBySeq(seq, msg)
	} else {
		if seq-recvQueue.GetStSeq() >= recvQueue.Size() {
			//Invalid seq
			return
		}
		for i := recvQueue.GetEnSeq(); i != seq; i++ {
			recvQueue.PushBack(nil)
		}
		recvQueue.PushBack(msg)
	}
	if recvQueue.Length() > 0 {
		partialDataList := nmph.partialDataList
		for msgInQueue := recvQueue.Get(recvQueue.GetStSeq()).(*MultipackMsg); msgInQueue != nil; msgInQueue = recvQueue.Get(recvQueue.GetStSeq()).(*MultipackMsg) {

			if partialDataList.Len() == 0 {
				if msgInQueue.Info.PackSeq != 0 {
					//Invalid Seq
					continue
				}
				partialDataList.PushBack(msgInQueue)
			} else {
				lastPack := partialDataList.Back().Value.(*MultipackMsg)
				if msgInQueue.Info.PackSeq != lastPack.Info.PackSeq+1 {
					//Invalid seq, clear partialDataList
					for ; partialDataList.Len() > 0; partialDataList.Remove(partialDataList.Front()) {
					}
					continue
				}
				partialDataList.PushBack(msgInQueue)
				if msgInQueue.Info.PackSeq == msgInQueue.Info.SubPackCnt-1 {
					var data []byte //TODO: pre allocate slice
					for ; partialDataList.Len() > 0; partialDataList.Remove(partialDataList.Front()) {
						packdata := partialDataList.Front().Value.(*MultipackMsg).Data
						data = append(data, packdata...)
					}
					go func() {
						nmph.assembledChan <- data
					}()
				}
			}

			recvQueue.PopFront()
			if recvQueue.Length() == 0 {
				break
			}
		}
	}
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
		msg.State = nmph.getState()
		msg.Info.SendTime = time.Now().Unix()
		nmsg.Data = nmph.encoder.Encode(msg)
		duration := time.Duration(nmph.rto)
		if timer, ok := nmph.retransmitTimerMap[msg.Info.Seq]; ok {
			duration = timer.GetDuration() * 2
			timer.Stop()
		}
		timer := common.NewTimer(msg.Info.Seq, nmph.rootContext, nmph.retansmitChan)
		timer.Start(duration)
		nmph.retransmitTimerMap[msg.Info.Seq] = timer
		//TODO precise start time
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
		case seq := <-nmph.retansmitChan:
			msg := nmph.sendingQueue.GetBySeq(seq).(*MultipackMsg)
			nmph.calcCwnd(EVENT_TIMEOUT)
			nmph.send(msg)
		case <-nmph.ctrlChan:
			nmph.statusTimer.Stop()
			nmph.rootCancelFunc()
		}
	}
}
