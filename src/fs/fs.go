package fs

import (
	"sync"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type DataPack struct {
	Keys []string
	Data [][]byte
}

type FileShareInfo struct {
	Keys   []string
	Origin DataOrigin
	Peers  []common.NodeID
}

type FilePurchaseInfo struct {
	Keys []string
	Peer common.NodeID
	Hash common.HashVal
}

type FS interface {
	Set(string, []byte)
	Get(string) ([]byte, error)
	Share(string, *FileShareInfo)
	Purchase(string, chan interface{})
	Start()
}

type NaiveFS struct {
	db                    sync.Map
	sessionId             uint32
	staticAppliNetHandler net.AppliNetHandler
	encoder               common.Encoder

	shareChan           chan *FileShareInfo
	shareResultChan     chan *ShareResult
	shareRequestChan    chan *net.ListenerNetMsg
	shareRecvResultChan chan *ShareRecvResult
	purchaseChan        chan *FilePurchaseInfo
	purchaseResultChan  chan *PurchaseResult
	puchaseRequestChan  chan *net.ListenerNetMsg
	ctrlChan            chan struct{}
}

func NewNaiveFS() *NaiveFS {
	ret := &NaiveFS{}
	return ret
}

func (nfs *NaiveFS) Set(k string, v []byte) {
	nfs.db.Store(k, v)
}

func (nfs *NaiveFS) Get(k string) ([]byte, error) {
	ret, _ := nfs.db.Load(k)
	return ret.([]byte), nil
}

func (nfs *NaiveFS) Share(k string, info *FileShareInfo) {
	go func() {
		nfs.shareChan <- info
	}()
}

func (nfs *NaiveFS) Purchase(url string, c chan interface{}) {}

func (nfs *NaiveFS) Start() {
	nfs.staticAppliNetHandler.AddListener(SPROC_WAIT, nfs.shareRequestChan)
	nfs.staticAppliNetHandler.AddListener(PPROC_WAIT, nfs.puchaseRequestChan)
	go nfs.run()
}

func (nfs *NaiveFS) run() {
	for {
		select {
		case info := <-nfs.shareChan:
			nfs.sessionId += 1
			handler := &net.NaiveAppliNetHandler{}
			session := NewShareSession(nfs, nfs.sessionId, info.Keys, info.Origin, info.Peers, handler, nfs.encoder, nfs.shareResultChan)
			session.Start()
		case msg := <-nfs.shareRequestChan:
			var req ShareRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			handler := &net.NaiveAppliNetHandler{}
			session := NewShareRecvSession(nfs, req.ID, req.Hash, req.Size, msg.FromPeerID, msg.FromHandlerID, handler, nfs.encoder, nfs.shareRecvResultChan)
			session.Start()
		case <-nfs.shareResultChan:
			return
		case <-nfs.shareRecvResultChan:
			return
		case info := <-nfs.purchaseChan:
			nfs.sessionId += 1
			session := NewPurchaseSession(nfs, nfs.sessionId, info.Keys, info.Hash, info.Peer, nfs.encoder, nfs.purchaseResultChan)
			session.Start()
		case msg := <-nfs.puchaseRequestChan:
			var req PurchaseRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			handler := &net.NaiveAppliNetHandler{}
			session := NewSellSession(nfs, req.Keys, msg.FromPeerID, msg.FromHandlerID, handler, nfs.encoder)
			session.Start()
		case <-nfs.purchaseResultChan:
			return
		case <-nfs.ctrlChan:
			return
		}
	}
}
