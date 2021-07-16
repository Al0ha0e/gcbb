package fs

import (
	"fmt"
	"time"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type TrackerSessionState uint8

const (
	TRACKER_UNKOWN     TrackerSessionState = iota
	TRACKER_PURCHASING TrackerSessionState = iota
	TRACKER_PURCHASED  TrackerSessionState = iota
)

type TrackerRequestMsg struct {
	KeyGroup [][]string
}

type TrackerResultMsg struct {
	PeerGroup [][]common.NodeID
}

type ParalleledPurchaseResult struct {
	ID uint32
	OK bool
}

func NewParalleledPurchaseResult(id uint32, ok bool) *ParalleledPurchaseResult {
	return &ParalleledPurchaseResult{
		ID: id,
		OK: ok,
	}
}

type ParalleledPurchaseSession struct {
	fs             FS
	id             uint32
	request        *ParalleledPurchaseInfo
	peerGroup      []map[common.NodeID]struct{}
	fileState      []TrackerSessionState
	unpurchasedCnt uint32
	appliHandler   net.AppliNetHandler
	encoder        common.Encoder
	waitTimer      *time.Timer

	trackerResultChan  chan *net.ListenerNetMsg
	purchaseResultChan chan *PurchaseResult
	resultChan         chan *ParalleledPurchaseResult
	ctrlChan           chan struct{}
}

func NewParalleledPurchaseSession(fs FS,
	id uint32,
	request *ParalleledPurchaseInfo,
	appliHandler net.AppliNetHandler,
	encoder common.Encoder,
	resultChan chan *ParalleledPurchaseResult) *ParalleledPurchaseSession {
	l := len(request.KeyGroup)
	ret := &ParalleledPurchaseSession{
		fs:                 fs,
		id:                 id,
		request:            request,
		peerGroup:          make([]map[common.NodeID]struct{}, l),
		fileState:          make([]TrackerSessionState, l),
		unpurchasedCnt:     uint32(l),
		appliHandler:       appliHandler,
		encoder:            encoder,
		trackerResultChan:  make(chan *net.ListenerNetMsg, len(request.Trackers)),
		purchaseResultChan: make(chan *PurchaseResult, l),
		resultChan:         resultChan,
		ctrlChan:           make(chan struct{}, 1),
	}
	for i := 0; i < l; i++ {
		ret.peerGroup[i] = make(map[common.NodeID]struct{})
		ret.fileState[i] = TRACKER_UNKOWN
	}
	return ret
}

func (session *ParalleledPurchaseSession) Start() {
	session.appliHandler.AddListener(common.TPROC_RECEIVER, session.trackerResultChan)
	request := &TrackerRequestMsg{
		KeyGroup: session.request.KeyGroup,
	}
	msg := session.encoder.Encode(request)
	for _, peer := range session.request.Trackers {
		session.appliHandler.SendTo(peer, net.StaticHandlerID, common.TPROC_WAIT, msg)
	}
	var totSize uint32 = 0
	for _, size := range session.request.Sizes {
		totSize += size
	}
	session.waitTimer = time.NewTimer(3 * session.appliHandler.EstimateTimeOut(totSize))
	go session.run()
}

func (session *ParalleledPurchaseSession) updatePeers(peerGroup [][]common.NodeID) {
	fmt.Println("UPDATE PEERS", peerGroup)
	for i, peers := range peerGroup {
		purchased := false
		for _, peer := range peers {
			if !purchased {
				purchased = true
				session.purchase(i, peer)
			}
			session.peerGroup[i][peer] = struct{}{}
		}
	}
}

func (session *ParalleledPurchaseSession) purchase(id int, peer common.NodeID) {
	if session.fileState[id] == TRACKER_UNKOWN {
		session.fileState[id] = TRACKER_PURCHASING
		session.fs.Purchase(NewFilePurchaseInfo(
			session.request.KeyGroup[id],
			session.request.Sizes[id],
			peer,
			session.request.Hashes[id],
			session.purchaseResultChan))
	}
}

func (session *ParalleledPurchaseSession) terminate(ok bool) {
	session.waitTimer.Stop()
	go func() {
		session.resultChan <- NewParalleledPurchaseResult(session.id, ok)
	}()
}

func (session *ParalleledPurchaseSession) run() {
	for {
		select {
		case msg := <-session.trackerResultChan:
			var result TrackerResultMsg
			session.encoder.Decode(msg.Data, &result)
			if len(result.PeerGroup) == len(session.request.KeyGroup) {
				session.updatePeers(result.PeerGroup)
			}
		case result := <-session.purchaseResultChan:
			for i := 0; i < len(session.request.KeyGroup); i++ {
				if result.Keys[0] == session.request.KeyGroup[i][0] {
					delete(session.peerGroup[i], result.Peer)
					if result.OK {
						session.fileState[i] = TRACKER_PURCHASED
						session.unpurchasedCnt -= 1
						if session.unpurchasedCnt == 0 {
							session.terminate(true)
						}
					} else {
						session.fileState[i] = TRACKER_UNKOWN
						for peer, _ := range session.peerGroup[i] {
							session.purchase(i, peer)
							break
						}
					}
					break
				}
			}
		case <-session.waitTimer.C:
			session.terminate(false)
			return
		case <-session.ctrlChan:
			session.terminate(false)
			return
		}
	}
}
