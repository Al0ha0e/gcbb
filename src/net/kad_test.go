package net

import (
	"container/list"
	"fmt"
	"testing"
)

func TestNewKadDHT(t *testing.T) {
	t.Log("SDS")
	l := list.New()
	l.Init()
	l.PushFront(1)
	fmt.Println(l.Front().Value)
}
