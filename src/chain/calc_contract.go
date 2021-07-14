package chain

import "github.com/gcbb/src/common"

type CalcContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(result chan *DeployResult)
	GetAddress() (bool, common.ContractAddress)
	Submit(sign []byte, ansHash []common.HashVal, result chan *CallResult)
	Terminate(key [20]byte, result chan *CallResult)
	ListenSubmit(result chan *CallResult)
	ListenPunish(result chan *CallResult)
}
