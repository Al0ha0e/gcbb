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

type TaskFileMetaInfo struct {
	KeyGroup [][]string
	Sizes    []uint32
	Hashes   []HashVal
	Trackers []NodeID
}

func NewTaskFileMetaInfo(keyGroup [][]string,
	sizes []uint32,
	hashes []HashVal,
	trackers []NodeID) *TaskFileMetaInfo {
	return &TaskFileMetaInfo{
		KeyGroup: keyGroup,
		Sizes:    sizes,
		Hashes:   hashes,
		Trackers: trackers,
	}
}

type TaskMetaInfo struct {
	MasterID    NodeID
	TaskID      TaskID
	ConfoundKey [20]byte
	Sign        []byte
	FileInfo    TaskFileMetaInfo
}

func NewTaskMetaInfo(
	masterID NodeID,
	taskID TaskID,
	confoundKey [20]byte,
	sign []byte,
	fileInfo *TaskFileMetaInfo) *TaskMetaInfo {
	return &TaskMetaInfo{
		MasterID:    masterID,
		TaskID:      taskID,
		ConfoundKey: confoundKey,
		Sign:        sign,
		FileInfo:    *fileInfo,
	}
}
