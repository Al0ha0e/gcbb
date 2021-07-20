package common

type MasterReqMsg struct {
	MasterID     NodeID
	ContractAddr ContractAddress
	Code         []byte
}

func NewMasterReqMsg(id NodeID, addr ContractAddress, code []byte) *MasterReqMsg {
	return &MasterReqMsg{
		MasterID:     id,
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

type TaskExecuteInfo struct {
	InputKeys  [][]string
	OutputKeys []string
}

func NewTaskExecuteInfo(inputKeys [][]string, outputKeys []string) *TaskExecuteInfo {
	return &TaskExecuteInfo{
		InputKeys:  inputKeys,
		OutputKeys: outputKeys,
	}
}

type TaskMetaInfo struct {
	MasterID    NodeID
	TaskID      TaskID
	ConfoundKey [20]byte
	Sign        []byte
	FileInfo    TaskFileMetaInfo
	ExecuteInfo TaskExecuteInfo
}

func NewTaskMetaInfo(
	masterID NodeID,
	taskID TaskID,
	confoundKey [20]byte,
	sign []byte,
	fileInfo *TaskFileMetaInfo,
	executeInfo *TaskExecuteInfo) *TaskMetaInfo {
	return &TaskMetaInfo{
		MasterID:    masterID,
		TaskID:      taskID,
		ConfoundKey: confoundKey,
		Sign:        sign,
		FileInfo:    *fileInfo,
		ExecuteInfo: *executeInfo,
	}
}
