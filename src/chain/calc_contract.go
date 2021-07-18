package chain

import (
	"strconv"

	"github.com/gcbb/src/common"
)

type CalcContractDeployResult struct{}

type CalcContractHandler interface {
	Deploy(taskID common.TaskID, resultChan chan *DeployResult)
	Validate(resultChan chan *DeployResult)
	GetAddress() (bool, common.ContractAddress)
	Submit(sign []byte, ansHash []common.HashVal, resultChan chan *CallResult)
	Terminate(key [20]byte, resultChan chan *CallResult)
	ListenSubmit(resultChan chan *CallResult)
	ListenPunish(resultChan chan *CallResult)
	// ListenTerminate(resultChan chan *CallResult)
}

type NaiveCalcContractHandler struct {
	address common.ContractAddress
	id      common.NodeID
}

func NewNaiveCalcContractHandler(id common.NodeID) *NaiveCalcContractHandler {
	return &NaiveCalcContractHandler{
		id: id,
	}
}

var addressNum int

type CalcContractSimulator struct {
	DeployInfo      *DeployResult
	SubmitListeners map[common.NodeID]chan *CallResult
}

func NewCalcContractSimulator(deployInfo *DeployResult) *CalcContractSimulator {
	return &CalcContractSimulator{
		DeployInfo:      deployInfo,
		SubmitListeners: make(map[common.NodeID]chan *CallResult),
	}
}

func ResolveDeployArgs(args []interface{}) common.TaskID {
	return args[0].(common.TaskID)
}

var calcContracts map[common.ContractAddress]*CalcContractSimulator

func (handler *NaiveCalcContractHandler) Deploy(taskID common.TaskID, resultChan chan *DeployResult) {
	addressNum += 1
	s := strconv.Itoa(addressNum)
	bts := []byte(s)
	copy(handler.address[:], bts)
	resultInfo := NewDeployResult(handler.address, handler.id, true, []interface{}{taskID})
	if calcContracts == nil {
		calcContracts = make(map[common.ContractAddress]*CalcContractSimulator)
	}
	calcContracts[handler.address] = NewCalcContractSimulator(resultInfo)
	go func() {
		resultChan <- resultInfo
	}()
}

func (handler *NaiveCalcContractHandler) Validate(resultChan chan *DeployResult) {
	simu, ok := calcContracts[handler.address]
	var resultInfo *DeployResult
	if ok {
		resultInfo = simu.DeployInfo
	} else {
		resultInfo = &DeployResult{
			OK: false,
		}
	}
	go func() {
		resultChan <- resultInfo
	}()
}

func (handler *NaiveCalcContractHandler) GetAddress() (bool, common.ContractAddress) {
	return true, handler.address
}
func (handler *NaiveCalcContractHandler) Submit(sign []byte, ansHash []common.HashVal, resultChan chan *CallResult) {
	callResult := NewCallResult(true, handler.id, []interface{}{sign, ansHash})
	go func() {
		for _, listener := range calcContracts[handler.address].SubmitListeners {
			listener <- callResult
		}
		resultChan <- callResult
	}()
}
func (handler *NaiveCalcContractHandler) Terminate(key [20]byte, resultChan chan *CallResult) {
	callResult := NewCallResult(true, handler.id, []interface{}{key})
	go func() {
		resultChan <- callResult
	}()
}

func (handler *NaiveCalcContractHandler) ListenSubmit(resultChan chan *CallResult) {
	calcContracts[handler.address].SubmitListeners[handler.id] = resultChan
}

func (handler *NaiveCalcContractHandler) ListenPunish(resultChan chan *CallResult) {
}

type CalcContractHandlerFactory interface {
	GetCalcContractHandler() CalcContractHandler
}

type NaiveCalcContractHandlerFactory struct {
	id common.NodeID
}

func NewNaiveCalcContractHandlerFactory(id common.NodeID) *NaiveCalcContractHandlerFactory {
	return &NaiveCalcContractHandlerFactory{id: id}
}

func (factory *NaiveCalcContractHandlerFactory) GetCalcContractHandler() CalcContractHandler {
	return NewNaiveCalcContractHandler(factory.id)
}
