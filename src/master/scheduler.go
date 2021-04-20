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

func (sch *Scheduler) Schedule(task *Task) (fin bool) {
	return false
}

func (sch *Scheduler) Finalize(task *Task) {}

func (sch *Scheduler) Run() {
	for {
		select {
		case newTask := <-sch.TaskChan:
			newTask.Id = sch.TaskId
			sch.Tasks[sch.TaskId] = newTask
			sch.TaskId += 1
			sch.Schedule(newTask)
		case result := <-sch.ResultChan:
			if result.Code == STOK {
				task := sch.Tasks[result.TaskId]
				fin := sch.Schedule(task)
				if fin {
					//RETURN
				}
			}
		}
	}
}
