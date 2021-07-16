package common

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

type TaskMetaInfo struct {
	MasterID    NodeID
	TaskID      TaskID
	ConfoundKey [20]byte
	Sign        []byte
	KeyGroup    [][]string
	Sizes       []uint32
	Hashes      []HashVal
	Trackers    []NodeID
}

func NewTaskMetaInfo(
	masterID NodeID,
	taskID TaskID,
	confoundKey [20]byte,
	sign []byte,
	trackers []NodeID) *TaskMetaInfo {
	return &TaskMetaInfo{
		MasterID:    masterID,
		TaskID:      taskID,
		ConfoundKey: confoundKey,
		Sign:        sign,
		Trackers:    trackers,
	}
}
