package fs

import (
	"fmt"
	"testing"

	"github.com/gcbb/src/common"
	"github.com/gcbb/src/net"
)

// func TestFileShare(t *testing.T) {
// 	id1 := common.NodeID{1}
// 	id2 := common.NodeID{2}
// 	encoder := &common.NaiveNetEncoder{}

// 	emu1 := net.NewNetHandlerEmulator(id1, encoder)
// 	emu2 := net.NewNetHandlerEmulator(id2, encoder)
// 	emu1.Start()
// 	emu2.Start()

// 	hdl1 := net.NewNaiveAppliNetHandler(0, emu1)
// 	hdl2 := net.NewNaiveAppliNetHandler(0, emu2)

// 	fac1 := net.NewNaiveAppliNetHandlerFactory(emu1)
// 	fac2 := net.NewNaiveAppliNetHandlerFactory(emu2)

// 	fs1 := NewNaiveFS(hdl1, fac1, encoder)
// 	fs2 := NewNaiveFS(hdl2, fac2, encoder)

// 	fs1.Start()
// 	fs2.Start()

// 	fs1.Set("a", []byte{11, 45, 14})
// 	fs1.Set("b", []byte{19, 19, 81, 0})
// 	resultChan := make(chan *ShareResult, 1)

// 	info := &FileShareInfo{
// 		Keys:       []string{"a", "b"},
// 		Peers:      []common.NodeID{id2},
// 		ResultChan: resultChan,
// 	}
// 	fs1.Share(info)
// 	result := <-resultChan
// 	fmt.Println(result)
// 	// tmer := time.NewTimer(10 * time.Second)
// 	// <-tmer.C
// }

func TestFilePurchase(t *testing.T) {
	id1 := common.NodeID{1}
	id2 := common.NodeID{2}
	encoder := &common.NaiveNetEncoder{}

	emu1 := net.NewNetHandlerEmulator(id1, encoder)
	emu2 := net.NewNetHandlerEmulator(id2, encoder)
	emu1.Start()
	emu2.Start()

	hdl1 := net.NewNaiveAppliNetHandler(0, emu1)
	hdl2 := net.NewNaiveAppliNetHandler(0, emu2)

	fac1 := net.NewNaiveAppliNetHandlerFactory(emu1)
	fac2 := net.NewNaiveAppliNetHandlerFactory(emu2)

	fs1 := NewNaiveFS(hdl1, fac1, encoder)
	fs2 := NewNaiveFS(hdl2, fac2, encoder)

	fs1.Start()
	fs2.Start()

	fs1.Set("a", []byte{11, 45, 14})
	fs1.Set("b", []byte{19, 19, 81, 0})

	keys := []string{"a", "b"}

	totData := make([]byte, 0)
	for _, key := range keys {
		data, _ := fs1.Get(key)
		totData = append(totData, data...)
	}
	hash := common.GenSHA1(totData)
	resultChan := make(chan *PurchaseResult, 1)
	info := &FilePurchaseInfo{
		Keys:       keys,
		Size:       20,
		Peer:       id1,
		Hash:       hash,
		ResultChan: resultChan,
	}
	fs2.Purchase(info)
	result := <-resultChan
	fmt.Println(result)
}
