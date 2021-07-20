package worker

import (
	"fmt"
	"time"

	"github.com/gcbb/src/common"
)

type ExecuteResult struct {
	OK      bool
	AnsHash []common.HashVal
}

func NewExecuteResult(ok bool, ansHash []common.HashVal) *ExecuteResult {
	return &ExecuteResult{
		OK:      ok,
		AnsHash: ansHash,
	}
}

type Executer interface {
	Execute(code []byte, executeInfo *common.TaskExecuteInfo, resultChan chan *ExecuteResult)
}

type NaiveWASMExecuter struct{}

func NewNaiveWASMExecuter() *NaiveWASMExecuter {
	return &NaiveWASMExecuter{}
}

func (executer *NaiveWASMExecuter) Execute(code []byte, executeInfo *common.TaskExecuteInfo, resultChan chan *ExecuteResult) {
	go func() {
		tm := time.NewTimer(1 * time.Second)
		<-tm.C
		l := len(executeInfo.OutputKeys)
		fmt.Println("L", l)
		ansHash := make([]common.HashVal, l)
		for i := 0; i < l; i++ {
			ansHash[i] = common.HashVal{uint8(i + 1)}
		}
		resultChan <- NewExecuteResult(true, ansHash)
	}()
}

type ExecuterFactory interface {
	GetExecuter() Executer
}

type NaiveExecuterFactory struct{}

func NewNaiveExecuterFactory() *NaiveExecuterFactory {
	return &NaiveExecuterFactory{}
}

func (factory *NaiveExecuterFactory) GetExecuter() Executer {
	return NewNaiveWASMExecuter()
}
