package chain

import "github.com/gcbb/src/common"

type DeployResult struct {
	Address  common.ContractAddress
	Deployer common.NodeID
	OK       bool
	Args     []interface{}
}

func NewDeployResult(address common.ContractAddress,
	deployer common.NodeID,
	ok bool,
	args []interface{}) *DeployResult {
	return &DeployResult{
		Address:  address,
		Deployer: deployer,
		OK:       ok,
		Args:     args,
	}
}

type CallResult struct {
	OK     bool
	Caller common.NodeID
	Args   []interface{}
}

func NewCallResult(ok bool,
	caller common.NodeID,
	args []interface{}) *CallResult {
	return &CallResult{
		OK:     ok,
		Caller: caller,
		Args:   args,
	}
}

type ContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(address common.ContractAddress, result chan *DeployResult)
	GetAddress() (bool, common.ContractAddress)
	Call(method string, args []interface{}, result chan *CallResult)
	Listen(method string, result chan *CallResult)
}

// var

// type ContractHandlerSimulator struct {
// }

// func (*ContractHandlerSimulator) Deploy(args []interface{}, result chan *DeployResult) {

// }

// func (*ContractHandlerSimulator) Validate(address common.ContractAddress, result chan *DeployResult) {
// }
// func (*ContractHandlerSimulator) GetAddress() (bool, common.ContractAddress)                      {}
// func (*ContractHandlerSimulator) Call(method string, args []interface{}, result chan *CallResult) {}
// func (*ContractHandlerSimulator) Listen(method string, result chan *CallResult)                   {}
