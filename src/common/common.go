package common

import "math/big"

type ContractAddress [20]byte
type HashVal [20]byte
type NodeID [20]byte
type TaskID [20]byte

func BytesToBigInt(data []byte) *big.Int {
	ret := new(big.Int)
	ret.SetBytes(data)
	return ret
}
