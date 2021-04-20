package master

type UnitTask struct{

}

type SubTask struct{
	UnitTasks 	[]UnitTask
	PreRequests []uint
	PostTasks	[]uint
}

func NewSubTask() *SubTask{
	return &SubTask{}
}

type Task struct {
	SubTasks	[]*SubTask
	
}

func NewTask() *Task {
	return &Task{}
}
