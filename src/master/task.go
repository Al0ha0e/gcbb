package master

type UnitTask struct {
	DataUrl string
}

type SubTask struct {
	Code      []byte
	UnitTasks []UnitTask
	PreTasks  []uint32
	PostTasks []uint32
	InDeg     uint32
}

func NewSubTask() *SubTask {
	return &SubTask{}
}

type Task struct {
	Id       uint64
	SubTasks []*SubTask
}

func NewTask() *Task {
	return &Task{}
}
