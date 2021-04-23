package net

import (
	"container/list"

	"github.com/gcbb/src/common"
)

type DistributedHashTable interface {
	Insert(P2PHandler)
	Get(common.NodeID) P2PHandler
	Has(common.NodeID) bool
	Update(common.NodeID)
	Remove(common.NodeID)
	GetK() []P2PHandler
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
		KBucket: make([]*list.List, 64),
	}
	for i := 0; i < len(ret.KBucket); i++ {
		ret.KBucket[i] = list.New().Init()
	}
	return ret
}

func getKDist(id1 common.NodeID, id2 common.NodeID) int {
	return 0
}

func (dn *KadDHT) Insert(handler P2PHandler) {
	dist := getKDist(handler.GetDstId(), dn.Id)
	dn.KBucket[dist].PushFront(handler)
}

func (dn *KadDHT) Get(id common.NodeID) P2PHandler {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			return node.Value.(P2PHandler)
		}
	}
	return nil
}

func (dn *KadDHT) Has(id common.NodeID) bool {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			return true
		}
	}
	return false
}

func (dn *KadDHT) Update(id common.NodeID) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			dn.KBucket[dist].MoveToFront(node)
			return
		}
	}
}

func (dn *KadDHT) Remove(id common.NodeID) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetDstId() == id {
			dn.KBucket[dist].Remove(node)
			return
		}
	}
}

func (dn *KadDHT) GetK() []P2PHandler {
	ret := make([]P2PHandler, 1)
	return ret
}
