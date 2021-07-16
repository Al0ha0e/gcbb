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
// 	r1, _ := fs2.Get("a")
// 	r2, _ := fs2.Get("b")
// 	fmt.Println(result, r1, r2)
// 	// tmer := time.NewTimer(10 * time.Second)
// 	// <-tmer.C
// }

// func TestFilePurchase(t *testing.T) {
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

// 	keys := []string{"a", "b"}

// 	totData := make([]byte, 0)
// 	for _, key := range keys {
// 		data, _ := fs1.Get(key)
// 		totData = append(totData, data...)
// 	}
// 	hash := common.GenSHA1(totData)
// 	resultChan := make(chan *PurchaseResult, 1)
// 	info := &FilePurchaseInfo{
// 		Keys:       keys,
// 		Size:       20,
// 		Peer:       id1,
// 		Hash:       hash,
// 		ResultChan: resultChan,
// 	}
// 	fs2.Purchase(info)
// 	result := <-resultChan
// 	r1, _ := fs2.Get("a")
// 	r2, _ := fs2.Get("b")
// 	fmt.Println(result, r1, r2)
// }

func TestFileParalledPurchase(t *testing.T) {
	id1 := common.NodeID{1}
	id2 := common.NodeID{2}
	id3 := common.NodeID{3}
	id4 := common.NodeID{4}

	encoder := &common.NaiveNetEncoder{}

	emu1 := net.NewNetHandlerEmulator(id1, encoder)
	emu2 := net.NewNetHandlerEmulator(id2, encoder)
	emu3 := net.NewNetHandlerEmulator(id3, encoder)
	emu4 := net.NewNetHandlerEmulator(id4, encoder)
	emu1.Start()
	emu2.Start()
	emu3.Start()
	emu4.Start()

	hdl1 := net.NewNaiveAppliNetHandler(0, emu1)
	hdl2 := net.NewNaiveAppliNetHandler(0, emu2)
	hdl3 := net.NewNaiveAppliNetHandler(0, emu3)
	hdl4 := net.NewNaiveAppliNetHandler(0, emu4)

	fac1 := net.NewNaiveAppliNetHandlerFactory(emu1)
	fac2 := net.NewNaiveAppliNetHandlerFactory(emu2)
	fac3 := net.NewNaiveAppliNetHandlerFactory(emu3)
	fac4 := net.NewNaiveAppliNetHandlerFactory(emu4)

	fs1 := NewNaiveFS(hdl1, fac1, encoder)
	fs2 := NewNaiveFS(hdl2, fac2, encoder)
	fs3 := NewNaiveFS(hdl3, fac3, encoder)
	fs4 := NewNaiveFS(hdl4, fac4, encoder)

	fs1.Start()
	fs2.Start()
	fs3.Start()
	fs4.Start()

	fs1.Set("a", []byte{11, 45, 14})
	fs1.Set("b", []byte{19, 19, 81, 0})
	resultChan := make(chan *ShareResult, 1)
	resultChan2 := make(chan *ShareResult, 1)

	info := NewFileShareInfo([]string{"a"}, DataOrigin{}, []common.NodeID{id2}, resultChan)
	info2 := NewFileShareInfo([]string{"b"}, DataOrigin{}, []common.NodeID{id4}, resultChan2)

	fs1.Share(info)
	fs1.Share(info2)
	result := <-resultChan
	fmt.Println(result)
	<-resultChan2

	hash1 := common.GenSHA1([]byte{11, 45, 14})
	hash2 := common.GenSHA1([]byte{19, 19, 81, 0})

	resultChan3 := make(chan *ParalleledPurchaseResult, 1)
	pinfo := NewParalleledPurchaseInfo(make([][]string, 2),
		[]uint32{10, 10},
		[]common.HashVal{hash1, hash2},
		[]common.NodeID{id1},
		resultChan3)

	pinfo.KeyGroup[0] = []string{"a"}
	pinfo.KeyGroup[1] = []string{"b"}
	fs3.ParalleledPurchase(pinfo)

	result2 := <-resultChan3
	r1, _ := fs3.Get("a")
	r2, _ := fs3.Get("b")
	fmt.Println(result2, r1, r2)
}
