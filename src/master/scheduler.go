package master

type STResultCode uint8

const (
	STOK  STResultCode = iota
	STERR STResultCode = iota
)

type SubTaskResult struct {
	TaskId    uint64
	SubtaskId uint32
	Code      STResultCode
	Info      string
}

type Scheduler struct {
	Tasks      map[uint64]*Task
	TaskChan   chan *Task
	ResultChan chan *SubTaskResult
	TaskId     uint64
}

func NewScheduler(stTaskId uint64) *Scheduler {
	ret := &Scheduler{
		TaskId:     stTaskId,
		TaskChan:   make(chan *Task, 10),
		ResultChan: make(chan *SubTaskResult, 10),
	}
	return ret
}

func (sch *Scheduler) Schedule(task *Task, finishTask *SubTask) (fin bool) {
	if finishTask == nil {
		for _, subTask := range task.SubTasks {
			if subTask.InDeg == 0 {

			}
		}
	} else {
		for _, taskId := range finishTask.PostTasks {
			subTask := task.SubTasks[taskId]
			subTask.InDeg -= 1
			if subTask.InDeg == 0 {

			}
		}
		task.FinishCnt += 1
		if task.FinishCnt == uint32(len(task.SubTasks)) {
			return true
		}
	}
	return false
}

func (sch *Scheduler) InitalizeTask(task *Task) {

	for _, subTask := range task.SubTasks {
		if subTask.InDeg == 0 {

		}
	}
}

func (sch *Scheduler) FinalizeTask(task *Task) {}

func (sch *Scheduler) Run() {
	for {
		select {
		case newTask := <-sch.TaskChan:
			newTask.Id = sch.TaskId
			sch.Tasks[sch.TaskId] = newTask
			sch.TaskId += 1
			sch.InitalizeTask(newTask)
			sch.Schedule(newTask, nil)
		case result := <-sch.ResultChan:
			if result.Code == STOK {
				task := sch.Tasks[result.TaskId]
				fin := sch.Schedule(task, task.SubTasks[result.SubtaskId])
				if fin {
					sch.FinalizeTask(task)
				}
			}
		}
	}
}
