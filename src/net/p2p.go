package net

import (
	"net"
	"time"
	"encoding/json"
)

type PSMsgType	uint8

const(
	PS_CONN PSMsgType = iota
	PS_ROUTE PSMsgType = iota
)

type P2PHandlerState uint8
const (
	ST_WAIT P2PHandlerState = iota
	ST_PING P2PHandlerState = iota
	ST_CONN P2PHandlerState = iota
)

func Send(sock *net.UDPConn,data []byte,addr *net.UDPAddr){
	sock.WriteToUDP(data,addr)
}

func Recv(sock *net.UDPConn) chan *NetResult{
	ret := make(chan *NetResult,1)
	go func(result chan*NetResult){
		ret := &NetResult{}
		ret.L,ret.Addr,_ = sock.ReadFromUDP(ret.Data)
		result <- ret
	}(ret)
	return ret
}

type P2PServerMsg struct {
	Type PSMsgType  `json:"tp"`
	Data []byte `json:"data"`
}

type P2PConnMsg struct {
	SrcIP      	net.IP	`json:"sIP"`
	DstId		int64	`json:"dId"`
	HandlerPort int		`json:"hPort"`
}

type P2PServer interface {
	SetServer(*net.UDPAddr) chan *NetResult
	ConnectUDP() chan *NetResult
	Run()
}

type NaiveP2PServer struct {
	Id 			int64
	sock		*net.UDPConn
	Handlers	map[int64]*P2PHandler
	ConnResult	map[int64]chan *NetResult
	ServerAddr	*net.UDPAddr
	Connected bool
	// Handler Pool
}

func NewNaiveP2PServer(id int64,sock *net.UDPConn) *NaiveP2PServer {
	ret := &NaiveP2PServer{
		Id: id,
		sock:sock,
		Handlers: make(map[int64]*P2PHandler)
	}
	return ret
}


func (nps *NaiveP2PServer)SetServer(sAddr *net.UDPAddr) chan *NetResult{
	ret := make(chan *NetResult, 1)
	nps.ServerAddr = sAddr
	go nps.setServer(ret)
	return ret
}

func (nph *NaiveP2PServer) setServer(result chan *NetResult) {
	msg := &P2PServerMsg{
		Type: PS_CONN,
		Data: []byte(nph.Id),
	}
	payload,_ := json.Marshal(msg)
	severConnTimer := time.NewTicker(10 * time.Second)
	Send(nph.sock,nph.ServerAddr,payload)
	for {
		select {
		case <-severConnTimer.C:
			ret := &NetResult{Status: 1}
			result <- ret
		case <-Recv(nph.sock):
			nph.Connected = true
			ret := &NetResult{Status: 0}
			result <- ret
		}
	}
}

func (nps *NaiveP2PServer) ConnectUDP(id int64) chan *NetResult{
	if _,ok := nps.Handlers[id], ok{
		ret := make(chan *NetResult,1)
		ret <- &NetResult{
			Status: 1
		}
		return ret
	}
	//GET SOCKET
	handler := NewNaiveP2PHandler()
	handler.SetState(ST_WAIT)
	nps.Handlers[id] = handler
	connMsg := &P2PConnMsg{
		DstId: id,
	}
	msgPayload,_ := json.Marshal(connMsg)
	msg := &P2PServerMsg{
		Type: PS_ROUTE,
		Data: msgPayload,
	}
	payload ,_ := json.Marshal(msg)
	Send(nps.sock,nps.ServerAddr,payload)
	ret := make(chan *NetResult,1)
	nps.ConnResult[id] = ret
	return ret
}

func (nps *NaiveP2PServer) Run(){
	for {
		select{
		case msg := <- Recv(nps.sock):
			
		}
	}
}

type P2PHandler interface {
	Send([]byte, *net.UDPAddr)
	Recv() chan *NetResult
	SetState(P2PHandlerState)
	GetState()P2PHandlerState
	Ping() chan *NetResult
}

type NaiveP2PHandler struct {
	sock *net.UDPConn
	state P2PHandlerState
}

func NewNaiveP2PHandler(sock *net.UDPConn) *NaiveP2PHandler {
	return &NaiveP2PHandler{sock: sock}
}

func (nph *NaiveP2PHandler) SetState(state P2PHandlerState){
	nph.state = state
}

func (nph *NaiveP2PHandler) GetState() P2PHandlerState{
	return nph.state
}

func (nph *NaiveP2PHandler) Send(data []byte, addr *net.UDPAddr) {
	Send(nph.sock,data,addr)
}

func (nph *NaiveP2PHandler) Recv() chan *NetResult {
	return Recv(nph.sock)
}

func (nph *NaiveP2PHandler) Ping() chan *NetResult{

}