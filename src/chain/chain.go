package chain

import "github.com/gcbb/src/common"

type DeployResult struct {
	Address common.ContractAddress
	OK      bool
	Args    []interface{}
}

type CallResult struct {
	OK     bool
	Caller common.NodeID
	Args   []interface{}
}

type ContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(address common.ContractAddress, result chan *DeployResult)
	GetAddress() (bool, common.ContractAddress)
	Call(method string, args []interface{}, result chan *CallResult)
	Listen(method string, result chan *CallResult)
}
