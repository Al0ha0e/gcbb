package master

import (
	"github.com/gcbb/src/chain"
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

type STSessionState uint8

const (
	STASK_UN         STSessionState = iota
	STASK_DEPLOYING  STSessionState = iota
	STASK_RUNNING    STSessionState = iota
	STASK_TERMINATED STSessionState = iota
	STASK_ABORTED    STSessionState = iota
)

type STPeerState uint8

const (
	STPEER_ACCEPT    STPeerState = iota
	STPEER_RUNNING   STPeerState = iota
	STPEER_SUBMITTED STPeerState = iota
)

type PeerAnswerMsg struct {
	PeerID  common.NodeID
	AnsHash []common.HashVal
}

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
	taskInfo     *SubTask
	contractAddr chain.ContractAddress
	mid          common.NodeID
	key          [20]byte
	state        STSessionState
	trackers     *common.TrackerInfo
	peers        map[common.NodeID]*PeerTaskInfo
	answers      []map[common.HashVal]uint32

	encoder        common.Encoder
	handler        net.AppliNetHandler
	deployChan     chan chain.DeployResult
	peerResChan    chan *net.ListenerNetMsg
	peerAnswerChan chan *PeerAnswerMsg
	ctrlChan       chan struct{}
}

func NewSubTaskSession() *SubTaskSession {
	return nil
}

func (session *SubTaskSession) Start() {
	//TODO Deploy Contract
	go session.run()
}

func (session *SubTaskSession) publishTask() {
	session.state = STASK_RUNNING
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
					session.contractAddr = result.Address
					session.publishTask()
				} else {
					//TODO
					return
				}
			} else {
				//TODO
				return
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
					meta := common.NewTaskMetaMsg(session.mid, session.taskInfo.ID, confoundKey, sign, *session.trackers)
					session.handler.SendTo(resMsg.WorkerID, msg.FromHandlerID, common.CPROC_META, session.encoder.Encode(&meta))
					session.peers[resMsg.WorkerID].State = STPEER_RUNNING
				}
			} else {
				//TODO
			}
		case msg := <-session.peerAnswerChan:
			if info, ok := session.peers[msg.PeerID]; ok && info.State == STPEER_RUNNING {
				info.State = STPEER_SUBMITTED
				confoundKey := info.ConfoundKey
				for i, hash := range msg.AnsHash {
					answerInt := common.BytesToBigInt(hash[:])
					answerInt.Sub(answerInt, common.BytesToBigInt(confoundKey[:]))
					var answer common.HashVal
					copy(answer[:], answerInt.Bytes()[:20])
					cnt, has := session.answers[i][answer]
					if has {
						session.answers[i][answer] = cnt + 1
					} else {
						session.answers[i][answer] = 1
					}
					info.Answers = append(info.Answers, answer)
				}
				//TODO Check Termination
			} else {
				//TODO
			}
		case <-session.ctrlChan:
			return
		}
	}
}
