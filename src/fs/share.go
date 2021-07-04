package fs

import (
	"time"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

const (
	//
	SPROC_WAIT     common.AppliListenerID = iota
	SPROC_SENDER   common.AppliListenerID = iota
	SPROC_RECEIVER common.AppliListenerID = iota
)

type DataOrigin struct {
	//TODO
}

type ShareRequestMsg struct {
	Origin DataOrigin
	ID     uint32
	Hash   common.HashVal
	Size   uint32
}

func NewShareRequest(ori DataOrigin, id uint32, hash common.HashVal, size uint32) *ShareRequestMsg {
	return &ShareRequestMsg{
		Origin: ori,
		ID:     id,
		Hash:   hash,
		Size:   size,
	}
}

type ShareAcceptMsg struct {
	ID   uint32
	Hash common.HashVal
}

type ShareSessionState uint8

const (
	SHARE_ST       ShareSessionState = iota
	SHARE_SENDING  ShareSessionState = iota
	SHARE_FINISHED ShareSessionState = iota
	SHARE_ABORT    ShareSessionState = iota
)

type ShareResult struct {
	PeerStates []ShareSessionState
}

type ShareSession struct {
	fs           FS
	id           uint32
	keys         []string
	origin       DataOrigin
	peers        []common.NodeID
	peerState    map[common.NodeID]ShareSessionState
	appliHandler net.AppliNetHandler
	encoder      common.Encoder
	sendTimer    *time.Timer

	acceptMsgChan   chan *net.ListenerNetMsg
	sendResultChan  chan *net.SendResult
	shareResultChan chan<- *ShareResult
	ctrlChan        chan struct{}
}

func NewShareSession(fs FS,
	id uint32,
	keys []string,
	origin DataOrigin,
	peers []common.NodeID,
	appliHandler net.AppliNetHandler,
	encoder common.Encoder,
	shareResultChan chan<- *ShareResult) *ShareSession {
	peerState := make(map[common.NodeID]ShareSessionState)
	for _, peer := range peers {
		peerState[peer] = SHARE_ST
	}
	return &ShareSession{
		fs:              fs,
		id:              id,
		keys:            keys,
		origin:          origin,
		peers:           peers,
		peerState:       peerState,
		appliHandler:    appliHandler,
		encoder:         encoder,
		acceptMsgChan:   make(chan *net.ListenerNetMsg, 10),
		sendResultChan:  make(chan *net.SendResult, 10),
		shareResultChan: shareResultChan,
		ctrlChan:        make(chan struct{}, 1),
	}
}

func (session *ShareSession) Start() {
	dataSize := 0
	totData := make([]byte, 0)
	for _, key := range session.keys {
		data, _ := session.fs.Get(key)
		dataSize += len(data)
		totData = append(totData, data...)
	}
	hash := common.GenSHA1(totData)
	msg := NewShareRequest(session.origin, session.id, hash, uint32(dataSize))
	data := session.encoder.Encode(msg)
	for i := 0; i < len(session.peers); i++ {
		session.peerState[session.peers[i]] = SHARE_ST
		session.appliHandler.SendTo(session.peers[i], net.StaticHandlerID, SPROC_WAIT, data)
	}
	session.sendTimer = time.NewTimer(2 * session.appliHandler.EstimateTimeOut(msg.Size))
	go session.run()
}

func (session *ShareSession) terminate() {
	session.sendTimer.Stop()
	states := make([]ShareSessionState, len(session.peers))
	for i, peer := range session.peers {
		states[i] = session.peerState[peer]
	}

	go func() {
		session.shareResultChan <- &ShareResult{
			PeerStates: states,
		}
	}()
}

func (session *ShareSession) run() {
	for {
		select {
		case msg := <-session.acceptMsgChan:
			var acceptMsg ShareAcceptMsg
			session.encoder.Decode(msg.Data, &acceptMsg)
			peerID := msg.FromPeerID
			if state, ok := session.peerState[peerID]; ok && state == SHARE_ST {
				session.peerState[peerID] = SHARE_SENDING
				var id uint32
				for i := 0; i < len(session.peers); i++ {
					if session.peers[i] == peerID {
						id = uint32(i)
						break
					}
				}
				datas := make([][]byte, len(session.keys))
				for _, key := range session.keys {
					data, _ := session.fs.Get(key)
					datas = append(datas, data)
				}
				dataPack := &DataPack{
					Keys: session.keys,
					Data: datas,
				}
				session.appliHandler.ReliableSendTo(peerID, msg.FromHandlerID, SPROC_RECEIVER, session.encoder.Encode(&dataPack), id, session.sendResultChan)
			} else {
				//TODO
			}
		case result := <-session.sendResultChan:
			if result.OK {
				if state, ok := session.peerState[result.DstPeerID]; ok && state == SHARE_SENDING {
					session.peerState[result.DstPeerID] = SHARE_FINISHED
				}
			} else {
			}
		case <-session.sendTimer.C:
			session.terminate()
			return
		case <-session.ctrlChan:
			session.terminate()
			return
		}
	}
}

type ShareRecvResult struct {
	OK bool
}

type ShareRecvSession struct {
	fs              FS
	id              uint32
	hash            common.HashVal
	size            uint32
	senderID        common.NodeID
	senderHandlerID uint16
	appliHandler    net.AppliNetHandler
	encoder         common.Encoder
	waitTimer       *time.Timer

	resultChan chan<- *ShareRecvResult
	dataChan   chan *net.ListenerNetMsg
	ctrlChan   chan struct{}
}

func NewShareRecvSession(
	fs FS,
	id uint32,
	hash common.HashVal,
	size uint32,
	senderID common.NodeID,
	senderHandlerID uint16,
	appliHandler net.AppliNetHandler,
	encoder common.Encoder,
	resultChan chan<- *ShareRecvResult) *ShareRecvSession {
	return &ShareRecvSession{
		fs:              fs,
		id:              id,
		hash:            hash,
		size:            size,
		senderID:        senderID,
		senderHandlerID: senderHandlerID,
		appliHandler:    appliHandler,
		encoder:         encoder,
		resultChan:      resultChan,
		dataChan:        make(chan *net.ListenerNetMsg, 1),
		ctrlChan:        make(chan struct{}, 1),
	}
}

func (session *ShareRecvSession) Start() {
	session.appliHandler.AddListener(SPROC_RECEIVER, session.dataChan)
	msg := &ShareAcceptMsg{
		ID:   session.id,
		Hash: session.hash,
	}
	data := session.encoder.Encode(msg)
	session.appliHandler.SendTo(session.senderID, session.senderHandlerID, SPROC_SENDER, data)
	//TODO
	session.waitTimer = time.NewTimer(session.appliHandler.EstimateTimeOut(session.size))
	go session.run()
}

func (session *ShareRecvSession) Stop() {
	close(session.ctrlChan)
}

func (session *ShareRecvSession) terminate(ok bool) {
	session.waitTimer.Stop()
	go func() {
		session.resultChan <- &ShareRecvResult{OK: ok}
	}()
}

func (session *ShareRecvSession) run() {
	for {
		select {
		case data := <-session.dataChan:
			//TODO DataInfo
			var dataPack DataPack
			session.encoder.Decode(data.Data, &dataPack)
			totData := make([]byte, 0)
			for i, _ := range dataPack.Keys {
				totData = append(totData, dataPack.Data[i]...)
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
		case <-session.waitTimer.C:
			session.terminate(false)
			return
		case <-session.ctrlChan:
			session.terminate(false)
			return
		}
	}
}
