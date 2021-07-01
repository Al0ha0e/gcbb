package chain

type ContractAddress [20]byte

type DeployResult struct {
	Address ContractAddress
	OK      bool
}
