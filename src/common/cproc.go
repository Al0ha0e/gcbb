package common

type CalcProcListenerID uint16

const (
	//Worker -> Master
	CPROC_RES CalcProcListenerID = iota
	//Master -> Worker
	CPROC_META CalcProcListenerID = iota
)

type PeerResMsg struct {
	PeerID   NodeID
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
