package chain

type ContractAddress [20]byte

type DeployResult struct {
	Address ContractAddress
	OK      bool
}

type CallResult struct {
	OK   bool
	Args []interface{}
}

type ContractHandler interface {
	Deploy(args []interface{}, result chan *DeployResult)
	Validate(address ContractAddress, result chan *DeployResult)
	GetAddress() (bool, ContractAddress)
	Call(method string, args []interface{}, result chan *CallResult)
	Listen(method string, result chan *CallResult)
}
