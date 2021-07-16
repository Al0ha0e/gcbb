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
	TPROC_WAIT     common.AppliListenerID = iota
	TPROC_RECEIVER common.AppliListenerID = iota
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

type ParalleledPurchaseInfo struct {
	KeyGroup   [][]string
	Sizes      []uint32
	Hashes     []common.HashVal
	Trackers   []common.NodeID
	ResultChan chan *ParalleledPurchaseResult
}

type FileInfo struct {
	Owner common.NodeID
	Peers map[common.NodeID]struct{}
}

type PeerInfo struct {
	FilesFrom map[string]struct{}
	FilesTo   map[string]struct{}
}

type FS interface {
	Set(string, []byte)
	SetWithInfo(k string, v []byte, info *FileInfo)
	Get(string) ([]byte, error)
	Share(*FileShareInfo)
	Purchase(*FilePurchaseInfo)
	Start()
}

type NaiveFS struct {
	owner                  common.NodeID
	db                     sync.Map
	fileInfoDB             sync.Map
	peerInfoDB             sync.Map
	sessionId              uint32
	staticAppliNetHandler  net.AppliNetHandler
	appliNetHandlerFactory net.AppliNetHandlerFactory
	encoder                common.Encoder

	shareChan                         chan *FileShareInfo
	shareResultChan                   chan *ShareResult
	userShareResultChans              map[uint32]chan *ShareResult
	shareRequestChan                  chan *net.ListenerNetMsg
	shareRecvResultChan               chan *ShareRecvResult
	purchaseChan                      chan *FilePurchaseInfo
	purchaseResultChan                chan *PurchaseResult
	userPurchaseResultChans           map[uint32]chan *PurchaseResult
	puchaseRequestChan                chan *net.ListenerNetMsg
	paralleledPurchaseChan            chan *ParalleledPurchaseInfo
	paralleledPurchaseResultChan      chan *ParalleledPurchaseResult
	userParalleledPurchaseResultChans map[uint32]chan *ParalleledPurchaseResult
	trackerRequestChan                chan *net.ListenerNetMsg
	ctrlChan                          chan struct{}
}

func NewNaiveFS(handler net.AppliNetHandler, appliNetHandlerFactory net.AppliNetHandlerFactory, encoder common.Encoder) *NaiveFS {
	ret := &NaiveFS{
		sessionId:                         0,
		staticAppliNetHandler:             handler,
		appliNetHandlerFactory:            appliNetHandlerFactory,
		encoder:                           encoder,
		shareChan:                         make(chan *FileShareInfo, 10),
		shareResultChan:                   make(chan *ShareResult, 10),
		userShareResultChans:              make(map[uint32]chan *ShareResult),
		shareRequestChan:                  make(chan *net.ListenerNetMsg, 10),
		shareRecvResultChan:               make(chan *ShareRecvResult, 10),
		purchaseChan:                      make(chan *FilePurchaseInfo, 10),
		purchaseResultChan:                make(chan *PurchaseResult, 10),
		userPurchaseResultChans:           make(map[uint32]chan *PurchaseResult),
		puchaseRequestChan:                make(chan *net.ListenerNetMsg, 10),
		paralleledPurchaseChan:            make(chan *ParalleledPurchaseInfo, 10),
		paralleledPurchaseResultChan:      make(chan *ParalleledPurchaseResult, 10),
		userParalleledPurchaseResultChans: make(map[uint32]chan *ParalleledPurchaseResult),
		trackerRequestChan:                make(chan *net.ListenerNetMsg, 10),
		ctrlChan:                          make(chan struct{}, 1),
	}
	return ret
}

func (nfs *NaiveFS) SetWithInfo(k string, v []byte, info *FileInfo) {
	if _, ok := nfs.db.Load(k); ok {
		return
	}
	nfs.db.Store(k, v)
	nfs.fileInfoDB.Store(k, info)
}

func (nfs *NaiveFS) Set(k string, v []byte) {
	nfs.SetWithInfo(k, v, &FileInfo{Owner: nfs.owner, Peers: make(map[common.NodeID]struct{})})
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

func (nfs *NaiveFS) ParalleledPurchase(info *ParalleledPurchaseInfo) {
	go func() {
		nfs.paralleledPurchaseChan <- info
	}()
}

func (nfs *NaiveFS) Start() {
	nfs.staticAppliNetHandler.AddListener(SPROC_WAIT, nfs.shareRequestChan)
	nfs.staticAppliNetHandler.AddListener(PPROC_WAIT, nfs.puchaseRequestChan)
	nfs.staticAppliNetHandler.AddListener(TPROC_WAIT, nfs.trackerRequestChan)
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
			session := NewShareSession(nfs, nfs.sessionId, info, handler, nfs.encoder, nfs.shareResultChan)
			session.Start()
		case msg := <-nfs.shareRequestChan:
			var req ShareRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			fmt.Println("REQ", req)
			//TODO check origin
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewShareRecvSession(nfs, req.ID, req.Hash, req.Size, msg.FromPeerID, msg.FromHandlerID, handler, nfs.encoder, nfs.shareRecvResultChan)
			session.Start()
		case result := <-nfs.shareResultChan:
			go func() {
				nfs.userShareResultChans[result.ID] <- result
				delete(nfs.userShareResultChans, result.ID)
			}()
			for peer, state := range result.PeerStates {
				if state == SHARE_FINISHED {
					pinfo, ok := nfs.peerInfoDB.Load(peer)
					var peerInfo *PeerInfo
					if !ok {
						peerInfo = &PeerInfo{
							FilesFrom: make(map[string]struct{}),
							FilesTo:   make(map[string]struct{}),
						}
					} else {
						peerInfo = pinfo.(*PeerInfo)
					}

					for _, file := range result.Keys {
						finfo, _ := nfs.fileInfoDB.Load(file)
						fileInfo := finfo.(*FileInfo)
						fileInfo.Peers[peer] = struct{}{}
						nfs.fileInfoDB.Store(file, fileInfo)
						peerInfo.FilesTo[file] = struct{}{}
					}
					nfs.peerInfoDB.Store(peer, peerInfo)
				}
			}
			fmt.Println("SHARE RESULT", result)
		case result := <-nfs.shareRecvResultChan:
			if result.OK {
				pinfo, ok := nfs.peerInfoDB.Load(result.From)
				var peerInfo *PeerInfo
				if !ok {
					peerInfo = &PeerInfo{
						FilesFrom: make(map[string]struct{}),
						FilesTo:   make(map[string]struct{}),
					}
				} else {
					peerInfo = pinfo.(*PeerInfo)
				}
				for _, file := range result.Keys {
					peerInfo.FilesFrom[file] = struct{}{}
				}
				nfs.peerInfoDB.Store(result.From, peerInfo)
				//TODO Maintain Info
			}
			fmt.Println("SHARE RECV RESULT", result)
		case info := <-nfs.purchaseChan:
			nfs.sessionId += 1
			nfs.userPurchaseResultChans[nfs.sessionId] = info.ResultChan
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewPurchaseSession(nfs, nfs.sessionId, info, handler, nfs.encoder, nfs.purchaseResultChan)
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
		case info := <-nfs.paralleledPurchaseChan:
			nfs.sessionId += 1
			nfs.userParalleledPurchaseResultChans[nfs.sessionId] = info.ResultChan
			handler := nfs.appliNetHandlerFactory.GetHandler()
			session := NewParalleledPurchaseSession(nfs, nfs.sessionId, info, handler, nfs.encoder, nfs.paralleledPurchaseResultChan)
			session.Start()
		case result := <-nfs.paralleledPurchaseResultChan:
			go func() {
				nfs.userParalleledPurchaseResultChans[result.ID] <- result
				delete(nfs.userParalleledPurchaseResultChans, result.ID)
			}()
			fmt.Println("P_PURCHASE RESULT", result)
		case msg := <-nfs.trackerRequestChan:
			var req TrackerRequestMsg
			nfs.encoder.Decode(msg.Data, &req)
			fmt.Println("TRACKER REQUEST", req)
			res := &TrackerResultMsg{
				PeerGroup: make([][]common.NodeID, len(req.KeyGroup)),
			}
			for i, keys := range req.KeyGroup {
				peerMap := make(map[common.NodeID]struct{})
				for _, key := range keys {
					finfo, ok := nfs.fileInfoDB.Load(key)
					if ok {
						fileInfo := finfo.(*FileInfo)
						for peer, _ := range fileInfo.Peers {
							peerMap[peer] = struct{}{}
						}
					}
				}
				peers := make([]common.NodeID, 0, len(peerMap))
				for peer, _ := range peerMap {
					peers = append(peers, peer)
				}
				res.PeerGroup[i] = peers
			}
			fmt.Println("TRACKER INFO", res)
			nfs.staticAppliNetHandler.SendTo(msg.FromPeerID, msg.FromHandlerID, TPROC_RECEIVER, nfs.encoder.Encode(res))
		case <-nfs.ctrlChan:
			return
		}
	}
}
