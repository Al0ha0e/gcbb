package chain

import (
	"strconv"

	"github.com/gcbb/src/common"
)

type CalcContractDeployResult struct{}

type CalcContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(result chan *CalcContractDeployResult)
	GetAddress() (bool, common.ContractAddress)
	Submit(sign []byte, ansHash []common.HashVal, result chan *CallResult)
	Terminate(key [20]byte, result chan *CallResult)
	ListenSubmit(result chan *CallResult)
	ListenPunish(result chan *CallResult)
}

type NaiveCalcContractHandler struct {
	address common.ContractAddress
	id      common.NodeID
}

var addressNum int

type CalcContractSimulator struct {
	DeployInfo *DeployResult
	Listeners  map[common.NodeID]chan *CallResult
}

func NewCalcContractSimulator() *CalcContractSimulator {
	return &CalcContractSimulator{}
}

var calcContracts map[common.ContractAddress]*CalcContractSimulator

func (handler *NaiveCalcContractHandler) Deploy(args []interface{}, result chan *DeployResult) {
	resultInfo := &DeployResult{
		OK: true,
	}
	addressNum += 1
	s := strconv.Itoa(addressNum)
	bts := []byte(s)
	copy(handler.address[:], bts)
	if calcContracts == nil {
		calcContracts = make(map[common.ContractAddress]*CalcContractSimulator)
	}
	calcContracts[handler.address] = NewCalcContractSimulator()
	go func() {
		result <- resultInfo
	}()
}

func (handler *NaiveCalcContractHandler) Validate(result chan *DeployResult) {
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
		result <- resultInfo
	}()
}

func (handler *NaiveCalcContractHandler) GetAddress() (bool, common.ContractAddress) {
	return true, handler.address
}
func (handler *NaiveCalcContractHandler) Submit(sign []byte, ansHash []common.HashVal, result chan *CallResult) {
	go func() {
		for _, listener := range calcContracts[handler.address].Listeners {
			listener <- &CallResult{}
		}
	}()
}
func (handler *NaiveCalcContractHandler) Terminate(key [20]byte, result chan *CallResult) {
}
func (handler *NaiveCalcContractHandler) ListenSubmit(result chan *CallResult) {
	calcContracts[handler.address].Listeners[handler.id] = result
}
func (handler *NaiveCalcContractHandler) ListenPunish(result chan *CallResult) {

}
