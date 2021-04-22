package net

type DHTNode struct {
	Id      uint64
	Nxt     *DHTNode
	Handler *P2PHandler
}

type DistributedHashTable interface {
	Insert(*DHTNode)
	Update(uint64)
	Remove(uint64)
	GetK() []*DHTNode
}

type KadDHT struct {
	Id          uint64
	K           uint32
	KBucketHead []*DHTNode
}

func NewKadDHT(id uint64, k uint32) *KadDHT {
	return &KadDHT{
		Id:          id,
		K:           k,
		KBucketHead: make([]*DHTNode, 64),
	}
}

func getKDist(id1 uint64, id2 uint64) int {
	return 0
}

func (dn *KadDHT) Insert(node *DHTNode) {
	dist := getKDist(node.Id, dn.Id)
	node.Nxt = dn.KBucketHead[dist]
	dn.KBucketHead[dist] = node
}

func (dn *KadDHT) Update(id uint64) {
	dist := getKDist(id, dn.Id)
	pre := dn.KBucketHead[dist]
	if pre.Id == id {
		return
	}
	for ; pre.Nxt != nil; pre = pre.Nxt {
		if pre.Nxt.Id == id {
			node := pre.Nxt
			pre.Nxt = node.Nxt
			node.Nxt = dn.KBucketHead[dist]
			dn.KBucketHead[dist] = node
			break
		}
	}
}

func (dn *KadDHT) Remove(id uint64) {
	dist := getKDist(id, dn.Id)
	pre := dn.KBucketHead[dist]
	if pre.Id == id {
		dn.KBucketHead[dist] = pre.Nxt
	}
	for ; pre.Nxt != nil; pre = pre.Nxt {
		if pre.Nxt.Id == id {
			pre.Nxt = pre.Nxt.Nxt
			break
		}
	}
}

func (dn *KadDHT) GetK() []*DHTNode {
	ret := make([]*DHTNode, 1)
	return ret
}
