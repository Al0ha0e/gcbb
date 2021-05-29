package net

import (
	"container/list"
	"fmt"

	"github.com/gcbb/src/common"
)

type DistributedHashTable interface {
	Insert(P2PHandler)
	Get(common.NodeID) P2PHandler
	Has(common.NodeID) bool
	Update(common.NodeID)
	Remove(common.NodeID)
	GetK(id common.NodeID) []P2PHandler
}

type KadDHT struct {
	Id      common.NodeID
	K       uint32
	KBucket []*list.List
}

func NewKadDHT(id common.NodeID, k uint32) *KadDHT {
	ret := &KadDHT{
		Id:      id,
		K:       k,
		KBucket: make([]*list.List, 8*len(id)),
	}
	for i := 0; i < len(ret.KBucket); i++ {
		ret.KBucket[i] = list.New().Init()
	}
	return ret
}

func getKDist(id1 common.NodeID, id2 common.NodeID) int {
	var diff common.NodeID
	for i := 0; i < len(id1); i++ {
		diff[i] = id1[i] ^ id2[i]
	}
	for i := len(id1) - 1; i >= 0; i-- {
		diffByte := uint(diff[i])
		if diffByte > 0 {
			for j := 7; j >= 0; j-- {
				if diffByte&(1<<j) > 0 {
					return i*8 + j + 1
				}
			}
		}
	}
	return 0
}

func (dn *KadDHT) Insert(handler P2PHandler) {
	dist := getKDist(handler.GetDstId(), dn.Id)
	fmt.Println("INSERT", handler.GetDstId(), dist)
	dn.KBucket[dist].PushFront(handler)
}

func (dn *KadDHT) Get(id common.NodeID) P2PHandler {
	dist := getKDist(id, dn.Id)
	fmt.Println("GET", id, dist)
	for node := dn.KBucket[dist].Front(); node != nil; node = node.Next() {
		fmt.Println("TRY FIND", node.Value.(P2PHandler).GetDstId())
		if node.Value.(P2PHandler).GetDstId() == id {
			return node.Value.(P2PHandler)
		}
	}
	return nil
}

func (dn *KadDHT) Has(id common.NodeID) bool {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			return true
		}
	}
	return false
}

func (dn *KadDHT) Update(id common.NodeID) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			dn.KBucket[dist].MoveToFront(node)
			return
		}
	}
}

func (dn *KadDHT) Remove(id common.NodeID) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			dn.KBucket[dist].Remove(node)
			return
		}
	}
}

func (dn *KadDHT) GetK(id common.NodeID) []P2PHandler {
	ret := make([]P2PHandler, 0)
	for dist := getKDist(id, dn.Id); dist >= 0; dist-- {
		for node := dn.KBucket[dist].Front(); node != nil; node = node.Next() {
			ret = append(ret, node.Value.(P2PHandler))
			if len(ret) == int(dn.K) {
				return ret
			}
		}
	}
	return ret
}
