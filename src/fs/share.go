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
	ID     string
	Hash   common.HashVal
	Size   uint32
}

func NewShareRequest(ori DataOrigin, id string, hash common.HashVal) *ShareRequestMsg {
	return &ShareRequestMsg{
		Origin: ori,
		ID:     id,
		Hash:   hash,
	}
}

type ShareAcceptMsg struct {
	ID   string
	Hash common.HashVal
}

type ShareSessionState uint8

const (
	SHARE_ST       ShareSessionState = iota
	SHARE_SENDING  ShareSessionState = iota
	SHARE_FINISHED ShareSessionState = iota
	SHARE_ABORT    ShareSessionState = iota
)

type ShareSession struct {
	fs           FS
	dataID       string
	hash         common.HashVal
	origin       DataOrigin
	peers        []common.NodeID
	peerState    map[common.NodeID]ShareSessionState
	appliHandler net.AppliNetHandler
	encoder      common.Encoder

	acceptMsgChan  chan *net.ListenerNetMsg
	sendResultChan chan *net.SendResult
	ctrlChan       chan struct{}
	//TODO Timer
}

func NewShareSession() *ShareSession {
	return &ShareSession{}
}

func (session *ShareSession) Start() {
	msg := NewShareRequest(session.origin, session.dataID, session.hash)
	data := session.encoder.Encode(msg)
	for i := 0; i < len(session.peers); i++ {
		session.peerState[session.peers[i]] = SHARE_ST
		session.appliHandler.SendTo(session.peers[i], net.StaticHandlerID, SPROC_WAIT, data)
	}
	go session.run()
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
				data, _ := session.fs.Get(session.dataID)
				session.appliHandler.ReliableSendTo(peerID, msg.FromHandlerID, SPROC_RECEIVER, data, id, session.sendResultChan)
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
		case <-session.ctrlChan:
			return
		}
	}
}

type ShareRecvResult struct {
	OK bool
}

type ShareRecvSession struct {
	fs              FS
	dataID          string
	hash            common.HashVal
	size            uint32
	senderID        common.NodeID
	senderHandlerID uint16
	appliHandler    net.AppliNetHandler
	encoder         common.Encoder
	waitTimer       time.Timer

	resultChan chan<- *ShareRecvResult
	dataChan   chan *net.ListenerNetMsg
	ctrlChan   <-chan struct{}
}

func (session *ShareRecvSession) Start() {
	session.appliHandler.AddListener(SPROC_RECEIVER, session.dataChan)
	msg := &ShareAcceptMsg{
		ID:   session.dataID,
		Hash: session.hash,
	}
	data := session.encoder.Encode(msg)
	session.appliHandler.SendTo(session.senderID, session.senderHandlerID, SPROC_SENDER, data)
	//TODO
	session.waitTimer.Reset(session.appliHandler.EstimateTimeOut(session.size))
	go session.run()
}

func (session *ShareRecvSession) run() {
	for {
		select {
		case data := <-session.dataChan:
			//TODO DataInfo
			//TODO Check data
			session.waitTimer.Stop()
			session.fs.Set(session.dataID, data.Data)
			session.resultChan <- &ShareRecvResult{OK: true}
			return
		case <-session.waitTimer.C:
			session.resultChan <- &ShareRecvResult{OK: false}
			return
		case <-session.ctrlChan:
			session.waitTimer.Stop()
			session.resultChan <- &ShareRecvResult{OK: false}
			return
		}
	}
}
