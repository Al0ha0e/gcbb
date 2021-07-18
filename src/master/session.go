package master

import (
	"github.com/gcbb/src/chain"
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type STSessionState uint8

const (
	STASK_UN          STSessionState = iota
	STASK_DEPLOYING   STSessionState = iota
	STASK_RUNNING     STSessionState = iota
	STASK_TERMINATING STSessionState = iota
	STASK_TERMINATED  STSessionState = iota
	STASK_ABORTED     STSessionState = iota
)

type STPeerState uint8

const (
	STPEER_ACCEPT    STPeerState = iota
	STPEER_RUNNING   STPeerState = iota
	STPEER_SUBMITTED STPeerState = iota
)

type SubTaskSessionResult struct {
}

type PeerTaskInfo struct {
	ConfoundKey [20]byte
	State       STPeerState
	Answers     []common.HashVal
}

func NewPeerTaskInfo(key [20]byte, state STPeerState) *PeerTaskInfo {
	return &PeerTaskInfo{
		ConfoundKey: key,
		State:       state,
		Answers:     make([]common.HashVal, 1),
	}
}

type SubTaskSession struct {
	taskInfo        *SubTask
	mid             common.NodeID
	key             [20]byte
	state           STSessionState
	peers           map[common.NodeID]*PeerTaskInfo
	answers         []map[common.HashVal]uint32
	validAnswers    []common.HashVal
	validAnswerCnt  []uint32
	answerCnt       uint32
	contractHandler chain.CalcContractHandler
	encoder         common.Encoder
	appliNetHandler net.AppliNetHandler

	deployChan     chan *chain.DeployResult
	peerResChan    chan *net.ListenerNetMsg
	peerAnswerChan chan *chain.CallResult
	terminateChan  chan *chain.CallResult
	ctrlChan       chan struct{}
}

func NewSubTaskSession(taskInfo *SubTask,
	mid common.NodeID,
	contractHandler chain.CalcContractHandler,
	encoder common.Encoder,
	appliNetHandler net.AppliNetHandler) *SubTaskSession {
	//TODO GenKey
	unitTaskCnt := len(taskInfo.UnitTasks)
	ret := &SubTaskSession{
		taskInfo: taskInfo,
		mid:      mid,
		//key: ,
		state:           STASK_UN,
		peers:           make(map[common.NodeID]*PeerTaskInfo),
		answers:         make([]map[common.HashVal]uint32, unitTaskCnt),
		validAnswers:    make([]common.HashVal, unitTaskCnt),
		validAnswerCnt:  make([]uint32, unitTaskCnt),
		answerCnt:       0,
		contractHandler: contractHandler,
		encoder:         encoder,
		appliNetHandler: appliNetHandler,
		deployChan:      make(chan *chain.DeployResult, 1),
		peerResChan:     make(chan *net.ListenerNetMsg, 10),
		peerAnswerChan:  make(chan *chain.CallResult, 10),
		terminateChan:   make(chan *chain.CallResult, 1),
		ctrlChan:        make(chan struct{}, 1),
	}
	for i := 0; i < unitTaskCnt; i++ {
		ret.answers[i] = make(map[common.HashVal]uint32)
	}
	return ret
}

func (session *SubTaskSession) Start() {
	session.deployContract()
	go session.run()
}

func (session *SubTaskSession) deployContract() {
	session.contractHandler.Deploy(session.taskInfo.ID, session.deployChan)
	session.state = STASK_DEPLOYING
}

func (session *SubTaskSession) publishTask() {
	session.state = STASK_RUNNING
	_, address := session.contractHandler.GetAddress()
	request := common.NewMasterReqMsg(address, session.taskInfo.Code)
	session.appliNetHandler.Broadcast(common.CPROC_WAIT, session.encoder.Encode(request))
}

func (session *SubTaskSession) genConfoundKey(peerID common.NodeID) [20]byte {
	var data []byte
	data = append(data, session.mid[:]...)
	data = append(data, peerID[:]...)
	data = append(data, session.key[:]...)

	return common.GenSHA1(data)
}

func (session *SubTaskSession) genSign(peerID common.NodeID) []byte {
	return []byte{}
}

func (session *SubTaskSession) run() {
	for {
		select {
		case result := <-session.deployChan:
			if session.state == STASK_DEPLOYING {
				if result.OK {
					session.contractHandler.ListenSubmit(session.peerAnswerChan)
					session.publishTask()
				} else {
					//TODO
					return
				}
			}
		case msg := <-session.peerResChan:
			var resMsg common.WorkerResMsg
			session.encoder.Decode(msg.Data, &resMsg)
			if resMsg.WorkerID == msg.FromPeerID &&
				resMsg.MasterID == session.mid &&
				resMsg.TaskID == session.taskInfo.ID {
				if _, ok := session.peers[resMsg.WorkerID]; ok {
					//TODO
				} else {
					confoundKey := session.genConfoundKey(resMsg.WorkerID)
					session.peers[resMsg.WorkerID] = NewPeerTaskInfo(confoundKey, STPEER_ACCEPT)
					sign := session.genSign(resMsg.WorkerID)
					meta := common.NewTaskMetaInfo(session.mid, session.taskInfo.ID, confoundKey, sign, session.taskInfo.FileInfo)
					session.appliNetHandler.SendTo(resMsg.WorkerID, msg.FromHandlerID, common.CPROC_META, session.encoder.Encode(&meta))
					session.peers[resMsg.WorkerID].State = STPEER_RUNNING
				}
			} else {
				//TODO
			}
		case msg := <-session.peerAnswerChan:
			if info, ok := session.peers[msg.Caller]; ok && info.State == STPEER_RUNNING {
				info.State = STPEER_SUBMITTED
				confoundKey := info.ConfoundKey
				ansHash := msg.Args[1].([]common.HashVal)
				session.answerCnt += 1
				for i, hash := range ansHash {
					answerInt := common.BytesToBigInt(hash[:])
					answerInt.Sub(answerInt, common.BytesToBigInt(confoundKey[:]))
					var answer common.HashVal
					copy(answer[:], answerInt.Bytes()[:20])
					cnt, has := session.answers[i][answer]
					if has {
						cnt += 1
					} else {
						cnt = 1
					}
					session.answers[i][answer] = cnt
					info.Answers = append(info.Answers, answer)
					if session.validAnswerCnt[i] < cnt {
						session.validAnswers[i] = answer
						session.validAnswerCnt[i] = cnt
					}
				}
				//TODO Check Termination
				canTerminate := true
				for _, cnt := range session.validAnswerCnt {
					if cnt*2 <= session.answerCnt {
						canTerminate = false
						break
					}
				}
				if canTerminate {
					session.state = STASK_TERMINATING
					session.contractHandler.Terminate(session.key, session.terminateChan)
				}
			} else {
				//TODO
			}
		case <-session.terminateChan:
			session.state = STASK_TERMINATED
			return
		case <-session.ctrlChan:
			return
		}
	}
}
