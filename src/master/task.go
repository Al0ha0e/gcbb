package master

import "github.com/gcbb/src/common"

// type UnitTask struct {
// 	InputKey  []string
// 	OutputKey string
// }

// func NewUnitTask() *UnitTask {
// 	return &UnitTask{}
// }

type SubTask struct {
	ID           common.TaskID
	Code         []byte
	PreTasks     []uint32
	PostTasks    []uint32
	InDeg        uint32
	UnitTaskCnt  uint32
	MinAnswerCnt uint32
	FileInfo     *common.TaskFileMetaInfo
	ExecuteInfo  *common.TaskExecuteInfo
}

func NewSubTask(id common.TaskID,
	code []byte,
	preTasks []uint32,
	postTasks []uint32,
	inDeg uint32,
	unitTaskCnt uint32,
	minAnswerCnt uint32,
	fileInfo *common.TaskFileMetaInfo,
	executeInfo *common.TaskExecuteInfo) *SubTask {
	return &SubTask{
		ID:           id,
		Code:         code,
		PreTasks:     preTasks,
		PostTasks:    postTasks,
		InDeg:        inDeg,
		UnitTaskCnt:  unitTaskCnt,
		MinAnswerCnt: minAnswerCnt,
		FileInfo:     fileInfo,
		ExecuteInfo:  executeInfo,
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
