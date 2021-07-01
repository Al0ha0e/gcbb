package worker

import (
	"github.com/gcbb/src/chain"
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type WorkerSessionState uint8

const (
	WS_INITED    WorkerSessionState = iota //After Req
	WS_PREPARING WorkerSessionState = iota //During validation
	WS_STARTED   WorkerSessionState = iota //After validation
	WS_FETCHING  WorkerSessionState = iota //After Meta
	WS_RUNNING   WorkerSessionState = iota //After fetched
	WS_UPLOADING WorkerSessionState = iota //After running
	WS_ABORTED   WorkerSessionState = iota
	WS_DONE      WorkerSessionState = iota
)

type WorkerSession struct {
	state        WorkerSessionState
	code         []byte
	data         [][]byte
	chainHandler chain.ContractHandler
	netHandler   net.AppliNetHandler
	masterID     common.NodeID
	workerID     common.NodeID
	taskID       common.TaskID
	confoundKey  [20]byte

	dstHandler uint16
	encoder    common.Encoder

	validateChan chan *chain.DeployResult
	metaChan     chan *net.ListenerNetMsg
	dataChan     chan [][]byte
	resultChan   chan []common.HashVal
	ctrlChan     chan struct{}
}

func (session *WorkerSession) Start() {
	//TODO Validation
	go session.run()
}

func (session *WorkerSession) checkSign(sign []byte) bool {
	return true
}

func (session *WorkerSession) run() {
	for {
		select {
		case result := <-session.validateChan:
			if session.state == WS_PREPARING {
				if result.OK {
					session.state = WS_STARTED
					//TODO Check Other
					msg := &common.WorkerResMsg{
						WorkerID: session.workerID,
						MasterID: session.masterID,
						TaskID:   session.taskID,
					}
					session.netHandler.SendTo(session.masterID, session.dstHandler, common.CPROC_RES, session.encoder.Encode(msg))
				} else {
					//TODO
				}
			} else {
				//TODO
			}
		case msg := <-session.metaChan:
			if session.state == WS_STARTED {
				var meta common.TaskMetaMsg
				session.encoder.Decode(msg.Data, &meta)

				if meta.MasterID == session.masterID &&
					meta.TaskID == session.taskID &&
					session.checkSign(meta.Sign) {
					session.confoundKey = meta.ConfoundKey
					session.state = WS_FETCHING
					//TODO Fetch Data
				}
			} else {
				//TODO
			}
		case data := <-session.dataChan:
			if session.state == WS_FETCHING {
				session.state = WS_RUNNING
				session.data = data
				//TODO RUN
			} else { //TODO
			}
		case <-session.resultChan:
			if session.state == WS_RUNNING {
				session.state = WS_UPLOADING
				//TODO upload
			} else {
				//TODO
			}
		case <-session.ctrlChan:
			return
		}
	}
}
