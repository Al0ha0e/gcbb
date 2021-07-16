package worker

import (
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/fs"
	"github.com/gcbb/src/net"
)

type Worker struct {
	id                     common.NodeID
	fs                     fs.FS
	staticAppliNetHandler  net.AppliNetHandler
	appliNetHandlerFactory net.AppliNetHandlerFactory
	encoder                common.Encoder

	taskRequestChan chan *net.ListenerNetMsg
	ctrlChan        chan struct{}
}

func (worker *Worker) Start() {
	worker.staticAppliNetHandler.AddListener(common.CPROC_WAIT, worker.taskRequestChan)
	go worker.run()
}

func (worker *Worker) run() {
	for {
		select {
		case msg := <-worker.taskRequestChan:
			var req common.MasterReqMsg
			worker.encoder.Decode(msg.Data, &req)
			handler := worker.appliNetHandlerFactory.GetHandler()
			session := NewWorkerSession(
				worker.id,
				worker.fs,
				req.Code,
				msg.FromHandlerID,
				handler,
				worker.encoder)
			session.Start()
		case <-worker.ctrlChan:
			return
		}
	}
}
