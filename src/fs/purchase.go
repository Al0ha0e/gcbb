package fs

import (
	"fmt"
	"time"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type PurchaseRequestMsg struct {
	ID   uint32
	Keys []string
}

type PurchaseResult struct {
	ID   uint32
	OK   bool
	Peer common.NodeID
	Keys []string
}

func NewPurchaseResult(id uint32, ok bool, peer common.NodeID, keys []string) *PurchaseResult {
	return &PurchaseResult{
		ID:   id,
		OK:   ok,
		Peer: peer,
		Keys: keys,
	}
}

type PurchaseSession struct {
	fs           FS
	id           uint32
	purchaseInfo *FilePurchaseInfo
	appliHandler net.AppliNetHandler
	encoder      common.Encoder
	waitTimer    *time.Timer

	dataChan   chan *net.ListenerNetMsg
	resultChan chan *PurchaseResult
	ctrlChan   chan struct{}
}

func NewPurchaseSession(
	fs FS,
	id uint32,
	purchaseInfo *FilePurchaseInfo,
	appliHandler net.AppliNetHandler,
	encoder common.Encoder,
	resultChan chan *PurchaseResult) *PurchaseSession {
	return &PurchaseSession{
		fs:           fs,
		id:           id,
		purchaseInfo: purchaseInfo,
		appliHandler: appliHandler,
		encoder:      encoder,
		dataChan:     make(chan *net.ListenerNetMsg, 1),
		resultChan:   resultChan,
		ctrlChan:     make(chan struct{}, 1),
	}
}

func (session *PurchaseSession) Start() {
	session.appliHandler.AddListener(common.PPROC_RECEIVER, session.dataChan)
	msg := &PurchaseRequestMsg{
		ID:   session.id,
		Keys: session.purchaseInfo.Keys,
	}
	data := session.encoder.Encode(msg)
	session.appliHandler.SendTo(session.purchaseInfo.Peer, net.StaticHandlerID, common.PPROC_WAIT, data)
	session.waitTimer = time.NewTimer(2 * session.appliHandler.EstimateTimeOut(session.purchaseInfo.Size))
	go session.run()
}

func (session *PurchaseSession) Stop() {
	close(session.ctrlChan)
}

func (session *PurchaseSession) terminate(ok bool) {
	session.waitTimer.Stop()
	go func() {
		session.resultChan <- NewPurchaseResult(session.id, ok, session.purchaseInfo.Peer, session.purchaseInfo.Keys)
	}()
}

func (session *PurchaseSession) run() {
	for {
		select {
		case msg := <-session.dataChan:
			if msg.FromPeerID == session.purchaseInfo.Peer {
				var dataPack DataPack
				session.encoder.Decode(msg.Data, &dataPack)
				fmt.Println(dataPack)
				totData := make([]byte, 0)
				ok := true
				for i, key := range dataPack.Keys {
					totData = append(totData, dataPack.Data[i]...)
					if key != session.purchaseInfo.Keys[i] {
						ok = false
						break
					}
				}
				if !ok {
					session.terminate(false)
					return
				}
				hash := common.GenSHA1(totData)
				if hash == session.purchaseInfo.Hash {
					for i, key := range dataPack.Keys {
						session.fs.Set(key, dataPack.Data[i])
					}
					session.terminate(true)
				} else {
					fmt.Println("BAD HASH", hash, session.purchaseInfo.Hash)
					session.terminate(false)
				}
				return
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

type SellSession struct {
	// id                uint32
	fs                FS
	keys              []string
	receiverID        common.NodeID
	receiverHandlerID uint16
	appliHandler      net.AppliNetHandler
	encoder           common.Encoder
	sendTimer         *time.Timer

	sendResultChan chan *net.SendResult
	ctrlChan       chan struct{}
}

func NewSellSession(
	fs FS,
	keys []string,
	receiverID common.NodeID,
	receiverHandlerID uint16,
	appliHandler net.AppliNetHandler,
	encoder common.Encoder) *SellSession {
	return &SellSession{
		fs:                fs,
		keys:              keys,
		receiverID:        receiverID,
		receiverHandlerID: receiverHandlerID,
		appliHandler:      appliHandler,
		encoder:           encoder,
		sendResultChan:    make(chan *net.SendResult, 1),
		ctrlChan:          make(chan struct{}, 1),
	}
}

func (session *SellSession) Start() {
	datas := make([][]byte, 0, len(session.keys))
	dataSize := 0
	for _, key := range session.keys {
		data, _ := session.fs.Get(key)
		dataSize += len(data)
		datas = append(datas, data)
	}
	dataPack := &DataPack{
		Keys: session.keys,
		Data: datas,
	}
	session.appliHandler.ReliableSendTo(session.receiverID, session.receiverHandlerID, common.PPROC_RECEIVER, session.encoder.Encode(&dataPack), 0, session.sendResultChan)
	session.sendTimer = time.NewTimer(session.appliHandler.EstimateTimeOut(uint32(dataSize)))
	go session.run()
}

func (session *SellSession) Stop() {
	close(session.ctrlChan)
}

func (session *SellSession) run() {
	for {
		select {
		case result := <-session.sendResultChan:
			if result.OK {
				return
			} else {
				return
			}
		case <-session.sendTimer.C:
			return
		case <-session.ctrlChan:
			return
		}
	}
}
