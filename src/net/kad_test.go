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
	id1 := common.NodeID{0, 1, 0}
	id2 := common.NodeID{0, 4, 0}
	dist := getKDist(id1, id2)
	t.Log(id1)
	t.Log(id2)
	t.Log("DIST", dist)
}

func TestInsert(t *testing.T) {
	id1 := common.NodeID{0, 1, 0}
	id2 := common.NodeID{0, 4, 9}
	var dht DistributedHashTable = NewKadDHT(id1, 10)
	handler := &NaiveP2PHandler{
		dstId: id2,
	}
	dht.Insert(handler)
	t.Log(dht.Get(id2).GetDstId())
}
