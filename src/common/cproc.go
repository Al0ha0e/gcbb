package common

type AppliListenerID uint16

const (
	//Worker -> Master
	CPROC_RES AppliListenerID = iota
	//Master -> Worker
	CPROC_META AppliListenerID = iota
)

type MasterReqMsg struct {
	ContractAddr ContractAddress
	Code         []byte
}

func NewMasterReqMsg(addr ContractAddress, code []byte) *MasterReqMsg {
	return &MasterReqMsg{
		ContractAddr: addr,
		Code:         code,
	}
}

type WorkerResMsg struct {
	WorkerID NodeID
	MasterID NodeID
	TaskID   TaskID
}

type TrackerInfo struct {
}

type TaskMetaMsg struct {
	MasterID    NodeID
	TaskID      TaskID
	ConfoundKey [20]byte
	Sign        []byte
	Trackers    TrackerInfo
}

func NewTaskMetaMsg(
	masterID NodeID,
	taskID TaskID,
	confoundKey [20]byte,
	sign []byte,
	trackers TrackerInfo) *TaskMetaMsg {
	return &TaskMetaMsg{
		MasterID:    masterID,
		TaskID:      taskID,
		ConfoundKey: confoundKey,
		Sign:        sign,
		Trackers:    trackers,
	}
}
