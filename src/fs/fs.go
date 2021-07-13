package fs

import (
	"fmt"
	"sync"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

const (
	//
	SPROC_WAIT     common.AppliListenerID = iota
	SPROC_SENDER   common.AppliListenerID = iota
	SPROC_RECEIVER common.AppliListenerID = iota
	PPROC_WAIT     common.AppliListenerID = iota
	PPROC_RECEIVER common.AppliListenerID = iota
)

type DataPack struct {
	Keys []string
	Data [][]byte
}

type FileShareInfo struct {
	Keys       []string
	Origin     DataOrigin
	Peers      []common.NodeID
	ResultChan chan *ShareResult
}

type FilePurchaseInfo struct {
	Keys       []string
	Size       uint32
	Peer       common.NodeID
	Hash       common.HashVal
	ResultChan chan *PurchaseResult
}

type FS interface {
	Set(string, []byte)
	Get(string) ([]byte, error)
	Share(*FileShareInfo)
	Purchase(*FilePurchaseInfo)
	Start()
}

type NaiveFS struct {
	db                     sync.Map
	sessionId              uint32
	staticAppliNetHandler  net.AppliNetHandler
	appliNetHandlerFactory net.AppliNetHandlerFactory
	encoder                common.Encoder

	shareChan               chan *FileShareInfo
	shareResultChan         chan *ShareResult
	userShareResultChans    map[uint32]chan *ShareResult
	shareRequestChan        chan *net.ListenerNetMsg
	shareRecvResultChan     chan *ShareRecvResult
	purchaseChan            chan *FilePurchaseInfo
	purchaseResultChan      chan *PurchaseResult
	userPurchaseResultChans map[uint32]chan *PurchaseResult
	puchaseRequestChan      chan *net.ListenerNetMsg
	ctrlChan                chan struct{}
}

func NewNaiveFS(handler net.AppliNetHandler, appliNetHandlerFactory net.AppliNetHandlerFactory, encoder common.Encoder) *NaiveFS {
	ret := &NaiveFS{
		staticAppliNetHandler:   handler,
		appliNetHandlerFactory:  appliNetHandlerFactory,
		encoder:                 encoder,
		shareChan:               make(chan *FileShareInfo, 10),
		shareResultChan:         make(chan *ShareResult, 10),
		userShareResultChans:    make(map[uint32]chan *ShareResult),
		shareRequestChan:        make(chan *net.ListenerNetMsg, 10),
		shareRecvResultChan:     make(chan *ShareRecvResult, 10),
		purchaseChan:            make(chan *FilePurchaseInfo, 10),
		purchaseResultChan:      make(chan *PurchaseResult, 10),
		userPurchaseResultChans: make(map[uint32]chan *PurchaseResult),
		puchaseRequestChan:      make(chan *net.ListenerNetMsg, 10),
		ctrlChan:                make(chan struct{}, 1),
	}
	return ret
}

func (nfs *NaiveFS) Set(k string, v []byte) {
	nfs.db.Store(k, v)
}

func (nfs *NaiveFS) Get(k string) ([]byte, error) {
	ret, _ := nfs.db.Load(k)
	return ret.([]byte), nil
}

func (nfs *NaiveFS) Share(info *FileShareInfo) {
	go func() {
		nfs.shareChan <- info
	}()
}

func (nfs *NaiveFS) Purchase(info *FilePurchaseInfo) {
	go func() {
		nfs.purchaseChan <- info
	}()
}

func (nfs *NaiveFS) Start() {
	nfs.staticAppliNetHandler.AddListener(SPROC_WAIT, nfs.shareRequestChan)
	nfs.staticAppliNetHandler.AddListener(PPROC_WAIT, nfs.puchaseRequestChan)
	go nfs.run()
}

func (nfs *NaiveFS) Stop() {
	close(nfs.ctrlChan)
}

func (nfs *NaiveFS) run() {
	for {
		select {
		case info := <-nfs.shareChan:
			nfs.sessionId += 1
			nfs.userShareResultChans[nfs.sessionId] = info.ResultChan
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewShareSession(nfs, nfs.sessionId, info.Keys, info.Origin, info.Peers, handler, nfs.encoder, nfs.shareResultChan)
			session.Start()
		case msg := <-nfs.shareRequestChan:
			var req ShareRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			fmt.Println("REQ", req)
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewShareRecvSession(nfs, req.ID, req.Hash, req.Size, msg.FromPeerID, msg.FromHandlerID, handler, nfs.encoder, nfs.shareRecvResultChan)
			session.Start()
		case result := <-nfs.shareResultChan:
			go func() {
				nfs.userShareResultChans[result.ID] <- result
				delete(nfs.userShareResultChans, result.ID)
			}()
			fmt.Println("SHARE RESULT", result)
		case result := <-nfs.shareRecvResultChan:
			fmt.Println("SHARE RECV RESULT", result)
		case info := <-nfs.purchaseChan:
			nfs.sessionId += 1
			nfs.userPurchaseResultChans[nfs.sessionId] = info.ResultChan
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewPurchaseSession(nfs, nfs.sessionId, info.Size, info.Keys, info.Hash, info.Peer, handler, nfs.encoder, nfs.purchaseResultChan)
			session.Start()
		case msg := <-nfs.puchaseRequestChan:
			var req PurchaseRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewSellSession(nfs, req.Keys, msg.FromPeerID, msg.FromHandlerID, handler, nfs.encoder)
			session.Start()
		case result := <-nfs.purchaseResultChan:
			go func() {
				nfs.userPurchaseResultChans[result.ID] <- result
				delete(nfs.userPurchaseResultChans, result.ID)
			}()
			fmt.Println("PURCHASE RESULT", result)
		case <-nfs.ctrlChan:
			return
		}
	}
}
