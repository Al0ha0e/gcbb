package master

import (
	"fmt"
	"testing"
	"time"

	"github.com/gcbb/src/chain"
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/fs"
	"github.com/gcbb/src/net"
	"github.com/gcbb/src/worker"
)

func TestMasterSession(t *testing.T) {
	id1 := common.NodeID{1}
	id2 := common.NodeID{2}
	id3 := common.NodeID{3}
	encoder := &common.NaiveNetEncoder{}
	emu1 := net.NewNetHandlerEmulator(id1, encoder)
	emu2 := net.NewNetHandlerEmulator(id2, encoder)
	emu3 := net.NewNetHandlerEmulator(id3, encoder)
	emu1.Start()
	emu2.Start()
	emu3.Start()
	hdl1 := net.NewNaiveAppliNetHandler(0, emu1)
	hdl2 := net.NewNaiveAppliNetHandler(0, emu2)
	hdl3 := net.NewNaiveAppliNetHandler(0, emu3)
	fac1 := net.NewNaiveAppliNetHandlerFactory(emu1)
	fac2 := net.NewNaiveAppliNetHandlerFactory(emu2)
	fac3 := net.NewNaiveAppliNetHandlerFactory(emu3)
	fs1 := fs.NewNaiveFS(hdl1, fac1, encoder)
	fs2 := fs.NewNaiveFS(hdl2, fac2, encoder)
	fs3 := fs.NewNaiveFS(hdl3, fac3, encoder)
	fs1.Start()
	fs2.Start()
	fs3.Start()

	fs1.Set("a", []byte{11, 45, 14})
	fs1.Set("b", []byte{19, 19, 81, 0})
	resultChan := make(chan *fs.ShareResult, 1)
	info := fs.NewFileShareInfo([]string{"a", "b"}, fs.DataOrigin{}, []common.NodeID{id2}, resultChan)
	fs1.Share(info)

	<-resultChan
	fmt.Println("OHHHHHHHHHHHHHHH")

	worker := worker.NewWorker(id3, fs3, hdl3, fac3, chain.NewNaiveCalcContractHandlerFactory(id3), worker.NewNaiveExecuterFactory(), encoder)
	worker.Start()

	keys := []string{"a", "b"}

	totData := make([]byte, 0)
	for _, key := range keys {
		data, _ := fs1.Get(key)
		totData = append(totData, data...)
	}
	hash := common.GenSHA1(totData)

	meta := common.NewTaskFileMetaInfo(make([][]string, 1),
		[]uint32{20},
		[]common.HashVal{hash},
		[]common.NodeID{id1})
	meta.KeyGroup[0] = []string{"a", "b"}

	exInfo := common.NewTaskExecuteInfo(make([][]string, 1), make([]string, 1))
	exInfo.InputKeys[0] = []string{"a", "b"}
	exInfo.OutputKeys[0] = "c"
	taskInfo := NewSubTask(common.TaskID{11, 4, 51, 4},
		[]byte{19, 19, 81, 0},
		[]uint32{},
		[]uint32{},
		0,
		1,
		2,
		meta,
		exInfo)
	master := NewSubTaskSession(taskInfo, id1, chain.NewNaiveCalcContractHandler(id1), encoder, hdl1)
	master.Start()

	tm := time.NewTimer(10 * time.Second)
	<-tm.C
}
