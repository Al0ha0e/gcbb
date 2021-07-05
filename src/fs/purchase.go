package fs

import (
	"time"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

const (
	PPROC_WAIT     common.AppliListenerID = iota
	PPROC_RECEIVER common.AppliListenerID = iota
)

type PurchaseRequestMsg struct {
	ID   uint32
	Keys []string
}

type PurchaseResult struct {
	OK bool
}

type PurchaseSession struct {
	fs        FS
	id        uint32
	keys      []string
	hash      common.HashVal
	peer      common.NodeID
	encoder   common.Encoder
	waitTimer *time.Timer

	dataChan   chan *net.ListenerNetMsg
	resultChan chan *PurchaseResult
	ctrlChan   chan struct{}
}

func NewPurchaseSession(
	fs FS,
	id uint32,
	keys []string,
	hash common.HashVal,
	peer common.NodeID,
	encoder common.Encoder,
	resultChan chan *PurchaseResult) *PurchaseSession {
	return &PurchaseSession{
		fs:         fs,
		id:         id,
		keys:       keys,
		hash:       hash,
		peer:       peer,
		encoder:    encoder,
		dataChan:   make(chan *net.ListenerNetMsg, 1),
		resultChan: resultChan,
		ctrlChan:   make(chan struct{}, 1),
	}
}

func (session *PurchaseSession) Start() {
	go session.run()
}

func (session *PurchaseSession) Stop() {
	close(session.ctrlChan)
}

func (session *PurchaseSession) terminate(ok bool) {
	session.waitTimer.Stop()
	go func() {
		session.resultChan <- &PurchaseResult{OK: ok}
	}()
}

func (session *PurchaseSession) run() {
	for {
		select {
		case msg := <-session.dataChan:
			if msg.FromPeerID == session.peer {
				var dataPack DataPack
				session.encoder.Decode(msg.Data, &dataPack)
				totData := make([]byte, 0)
				ok := true
				for i, key := range dataPack.Keys {
					totData = append(totData, dataPack.Data[i]...)
					if key != session.keys[i] {
						ok = false
						break
					}
				}
				if !ok {
					session.terminate(false)
					return
				}
				hash := common.GenSHA1(totData)
				if hash == session.hash {
					for i, key := range dataPack.Keys {
						session.fs.Set(key, dataPack.Data[i])
					}
					session.terminate(true)
				} else {
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
	datas := make([][]byte, len(session.keys))
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
	session.appliHandler.ReliableSendTo(session.receiverID, session.receiverHandlerID, PPROC_RECEIVER, session.encoder.Encode(&dataPack), 0, session.sendResultChan)
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