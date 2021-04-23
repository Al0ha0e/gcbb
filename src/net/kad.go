package net

import "container/list"

type DistributedHashTable interface {
	Insert(P2PHandler)
	Update(uint64)
	Remove(uint64)
	GetK() []P2PHandler
}

type KadDHT struct {
	Id      uint64
	K       uint32
	KBucket []*list.List
}

func NewKadDHT(id uint64, k uint32) *KadDHT {
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

func getKDist(id1 uint64, id2 uint64) int {
	return 0
}

func (dn *KadDHT) Insert(handler P2PHandler) {
	dist := getKDist(handler.GetId(), dn.Id)
	dn.KBucket[dist].PushFront(handler)
}

func (dn *KadDHT) Update(id uint64) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetId() == id {
			dn.KBucket[dist].MoveToFront(node)
			return
		}
	}
}

func (dn *KadDHT) Remove(id uint64) {
	dist := getKDist(id, dn.Id)
	for node := dn.KBucket[dist].Front(); node.Next() != nil; node = node.Next() {
		if node.Value.(P2PHandler).GetId() == id {
			dn.KBucket[dist].Remove(node)
			return
		}
	}
}

func (dn *KadDHT) GetK() []P2PHandler {
	ret := make([]P2PHandler, 1)
	return ret
}
