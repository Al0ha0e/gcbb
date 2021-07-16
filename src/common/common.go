package common

import "math/big"

type AppliListenerID uint16

const (
	//
	SPROC_WAIT     AppliListenerID = iota
	SPROC_SENDER   AppliListenerID = iota
	SPROC_RECEIVER AppliListenerID = iota
	PPROC_WAIT     AppliListenerID = iota
	PPROC_RECEIVER AppliListenerID = iota
	TPROC_WAIT     AppliListenerID = iota
	TPROC_RECEIVER AppliListenerID = iota
	CPROC_WAIT     AppliListenerID = iota
	CPROC_RES      AppliListenerID = iota //Worker -> Master
	CPROC_META     AppliListenerID = iota //Master -> Worker
)

type ContractAddress [20]byte
type HashVal [20]byte
type NodeID [20]byte
type TaskID [20]byte

func BytesToBigInt(data []byte) *big.Int {
	ret := new(big.Int)
	ret.SetBytes(data)
	return ret
}
