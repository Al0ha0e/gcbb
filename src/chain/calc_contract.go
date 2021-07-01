package chain

type CalcContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(address ContractAddress, result chan *DeployResult)
	GetAddress() (bool, ContractAddress)
	Submit(result chan *CallResult)
	Terminate(result chan *CallResult)
	ListenSubmit(result chan *CallResult)
	ListenPunish(result chan *CallResult)
}
