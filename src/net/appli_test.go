package net

import (
	"fmt"
	"testing"
	"time"

	"github.com/gcbb/src/common"
)

func TestNewNetHandlerEmulator(t *testing.T) {
	id := common.NodeID{1}
	encoder := &common.NaiveNetEncoder{}
	emulator := NewNetHandlerEmulator(id, encoder)
	fmt.Println("SDUS", emulator)
}

func TestEmulatorSend(t *testing.T) {
	id1 := common.NodeID{1}
	id2 := common.NodeID{2}
	encoder := &common.NaiveNetEncoder{}

	emu1 := NewNetHandlerEmulator(id1, encoder)
	emu2 := NewNetHandlerEmulator(id2, encoder)

	go emu1.run()
	go emu2.run()

	msg1_2 := NewAppliNetMsg(1, 2, 2, []byte{1, 2, 3})
	msg2_1 := NewAppliNetMsg(2, 1, 1, []byte{3, 4, 5})

	emu1.SendTo(id2, msg1_2)
	emu2.SendTo(id1, msg2_1)

	a := time.NewTimer(1 * time.Second)
	<-a.C

}

func TestEmulatorReliableSend(t *testing.T) {
	id1 := common.NodeID{1}
	id2 := common.NodeID{2}
	encoder := &common.NaiveNetEncoder{}

	emu1 := NewNetHandlerEmulator(id1, encoder)
	emu2 := NewNetHandlerEmulator(id2, encoder)

	go emu1.run()
	go emu2.run()

	msg1_2 := NewAppliNetMsg(1, 2, 2, []byte{1, 2, 3})
	msg2_1 := NewAppliNetMsg(2, 1, 1, []byte{3, 4, 5})

	result1 := make(chan *SendResult, 1)
	result2 := make(chan *SendResult, 1)
	emu1.ReliableSendTo(id2, msg1_2, 8, result1)
	emu2.ReliableSendTo(id1, msg2_1, 9, result2)

	fmt.Println("result1", <-result1)
	fmt.Println("result2", <-result2)
}
