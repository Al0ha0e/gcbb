package master

import "github.com/gcbb/src/common"

type UnitTask struct {
	DataUrl string
}

func NewUnitTask() *UnitTask {
	return &UnitTask{}
}

type SubTask struct {
	ID        common.TaskID
	Code      []byte
	UnitTasks []*UnitTask
	PreTasks  []uint32
	PostTasks []uint32
	InDeg     uint32
	FileInfo  *common.TaskFileMetaInfo
}

func NewSubTask(id common.TaskID,
	code []byte,
	unitTasks []*UnitTask,
	preTasks []uint32,
	postTasks []uint32,
	inDeg uint32,
	fileInfo *common.TaskFileMetaInfo) *SubTask {
	return &SubTask{
		ID:        id,
		Code:      code,
		UnitTasks: unitTasks,
		PreTasks:  preTasks,
		PostTasks: postTasks,
		InDeg:     inDeg,
		FileInfo:  fileInfo,
	}
}

type Task struct {
	Id        uint64
	FinishCnt uint32
	SubTasks  []*SubTask
}

func NewTask() *Task {
	return &Task{}
}
