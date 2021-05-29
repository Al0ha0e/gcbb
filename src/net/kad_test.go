package net

import (
	"container/list"
	"fmt"
	"testing"

	"github.com/gcbb/src/common"
)

func TestNewKadDHT(t *testing.T) {
	t.Log("SDS")
	l := list.New()
	l.Init()
	l.PushFront(1)
	fmt.Println(l.Front().Value)
}

func TestGetKDist(t *testing.T) {
	ids := []common.NodeID{common.NodeID{1}, common.NodeID{2}, common.NodeID{3}}
	for i := 0; i < 3; i++ {
		id1 := ids[i]
		id2 := ids[(i+1)%3]
		t.Log(id1, id2, getKDist(id1, id2), getKDist(id2, id1))
	}
}

func TestInsert(t *testing.T) {
	id1 := common.NodeID{0, 1, 0}
	id2 := [20]byte{4}
	var dht DistributedHashTable = NewKadDHT(id1, 10)
	handler := &NaiveP2PHandler{
		dstId: id2,
	}
	dht.Insert(handler)
	t.Log(dht.Get(id2).GetDstId())
}
