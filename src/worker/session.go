package worker

import (
	"fmt"

	"github.com/gcbb/src/chain"
	"github.com/gcbb/src/common"
	"github.com/gcbb/src/fs"
	"github.com/gcbb/src/net"
)

type WorkerSessionState uint8

const (
	WS_INITED    WorkerSessionState = iota //After Req
	WS_PREPARING WorkerSessionState = iota //During validation
	WS_STARTED   WorkerSessionState = iota //After validation
	WS_FETCHING  WorkerSessionState = iota //After Meta
	WS_RUNNING   WorkerSessionState = iota //After fetched
	WS_UPLOADING WorkerSessionState = iota //After running
	WS_ABORTED   WorkerSessionState = iota
	WS_DONE      WorkerSessionState = iota
)

type WorkerSession struct {
	workerID        common.NodeID
	contractAddress common.ContractAddress
	fs              fs.FS
	state           WorkerSessionState
	code            []byte
	masterHandlerID uint16
	metaInfo        *common.TaskMetaInfo
	contractHandler chain.CalcContractHandler
	appliNetHandler net.AppliNetHandler
	encoder         common.Encoder

	validateChan       chan *chain.DeployResult
	metaChan           chan *net.ListenerNetMsg
	purchaseResultChan chan *fs.ParalleledPurchaseResult
	ctrlChan           chan struct{}
	//resultChan         chan []common.HashVal
}

func NewWorkerSession(
	workerID common.NodeID,
	masterID common.NodeID,
	contractAddress common.ContractAddress,
	fileSystem fs.FS,
	code []byte,
	masterHandlerID uint16,
	contractHandler chain.CalcContractHandler,
	appliNetHandler net.AppliNetHandler,
	encoder common.Encoder) *WorkerSession {
	ret := &WorkerSession{
		workerID:           workerID,
		contractAddress:    contractAddress,
		fs:                 fileSystem,
		state:              WS_INITED,
		code:               code,
		masterHandlerID:    masterHandlerID,
		metaInfo:           &common.TaskMetaInfo{},
		contractHandler:    contractHandler,
		appliNetHandler:    appliNetHandler,
		encoder:            encoder,
		validateChan:       make(chan *chain.DeployResult, 1),
		metaChan:           make(chan *net.ListenerNetMsg, 1),
		purchaseResultChan: make(chan *fs.ParalleledPurchaseResult, 1),
		ctrlChan:           make(chan struct{}, 1),
	}
	ret.metaInfo.MasterID = masterID
	return ret
}

func (session *WorkerSession) Start() {
	fmt.Println("WORKER START")
	session.contractHandler.Validate(session.contractAddress, session.validateChan)
	session.state = WS_PREPARING
	session.appliNetHandler.AddListener(common.CPROC_META, session.metaChan)
	go session.run()
}

func (session *WorkerSession) checkSign(sign []byte) bool {
	return true
}

func (session *WorkerSession) run() {
	for {
		select {
		case result := <-session.validateChan:
			fmt.Println("VALIDATE RESULT", result)
			if session.state == WS_PREPARING && result.OK && session.metaInfo.MasterID == result.Deployer {
				session.state = WS_STARTED
				session.metaInfo.TaskID = chain.ResolveDeployArgs(result.Args)
				msg := &common.WorkerResMsg{
					WorkerID: session.workerID,
					MasterID: session.metaInfo.MasterID,
					TaskID:   session.metaInfo.TaskID,
				}
				session.appliNetHandler.SendTo(session.metaInfo.MasterID, session.masterHandlerID, common.CPROC_RES, session.encoder.Encode(msg))
			} else {
				//TODO
			}
		case msg := <-session.metaChan:
			if session.state == WS_STARTED {
				var meta common.TaskMetaInfo
				session.encoder.Decode(msg.Data, &meta)
				fmt.Println("META!", meta)
				if meta.MasterID == session.metaInfo.MasterID &&
					meta.TaskID == session.metaInfo.TaskID &&
					session.checkSign(meta.Sign) {
					session.metaInfo = &meta
					session.state = WS_FETCHING
					info := fs.NewParalleledPurchaseInfo(
						meta.FileInfo,
						session.purchaseResultChan,
					)
					session.fs.ParalleledPurchase(info)
				}
			} else {
				//TODO
			}
		case <-session.purchaseResultChan:
			if session.state == WS_FETCHING {
				session.state = WS_RUNNING
				fmt.Println("PURCHASE OVER!")
				for _, k := range session.metaInfo.FileInfo.KeyGroup[0] {
					v, _ := session.fs.Get(k)
					fmt.Println(k, v)
				}
				//TODO RUN
			} else { //TODO
			}
		case <-session.ctrlChan:
			return
			// case <-session.resultChan:
			// 	if session.state == WS_RUNNING {
			// 		session.state = WS_UPLOADING
			// 		//TODO upload
			// 	} else {
			// 		//TODO
			// 	}
		}
	}
}
