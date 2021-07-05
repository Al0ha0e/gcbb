package net

import (
	"time"

	"github.com/gcbb/src/common"
)

const StaticHandlerID uint16 = 0

type AppliNetMsg struct {
	SrcHandlerID  uint16
	DstHandlerID  uint16
	DstListenerID common.AppliListenerID
	Data          []byte
}

func NewAppliNetMsg(srcHandlerID uint16, dstHandlerID uint16, listenerID common.AppliListenerID, data []byte) *AppliNetMsg {
	return &AppliNetMsg{
		SrcHandlerID:  srcHandlerID,
		DstHandlerID:  dstHandlerID,
		DstListenerID: listenerID,
		Data:          data,
	}
}

type ListenerNetMsg struct {
	FromPeerID    common.NodeID
	FromHandlerID uint16
	Data          []byte
}

type SendResult struct {
	ID            uint32
	OK            bool
	DstPeerID     common.NodeID
	DstHandlerID  uint16
	DstListenerID common.AppliListenerID
}

type AppliNetHandler interface {
	GetID() uint16
	AddListener(listenerID common.AppliListenerID, listenerChan chan *ListenerNetMsg)
	RemoveListener(listenerID common.AppliListenerID)
	SendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte)
	ReliableSendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte, id uint32, resultChan chan *SendResult)
	EstimateTimeOut(byteCnt uint32) time.Duration
	OnMsgArrive(from common.NodeID, msg *AppliNetMsg)
}

type NaiveAppliNetHandler struct {
	id        uint16
	listeners map[common.AppliListenerID]chan *ListenerNetMsg
}

func NewNaiveAppliNetHandler(id uint16) *NaiveAppliNetHandler {
	return &NaiveAppliNetHandler{
		id:        id,
		listeners: make(map[common.AppliListenerID]chan *ListenerNetMsg),
	}
}

func (handler *NaiveAppliNetHandler) GetID() uint16 { return handler.id }
func (handler *NaiveAppliNetHandler) AddListener(listenerID common.AppliListenerID, listenerChan chan *ListenerNetMsg) {
	handler.listeners[listenerID] = listenerChan
}
func (handler *NaiveAppliNetHandler) RemoveListener(listenerID common.AppliListenerID) {
	delete(handler.listeners, listenerID)
}

func (handler *NaiveAppliNetHandler) SendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte) {

}
func (handler *NaiveAppliNetHandler) ReliableSendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte, id uint32, resultChan chan *SendResult) {
}
func (handler *NaiveAppliNetHandler) EstimateTimeOut(byteCnt uint32) time.Duration {
	return time.Duration(byteCnt) * time.Second
}

func (handler *NaiveAppliNetHandler) OnMsgArrive(from common.NodeID, msg *AppliNetMsg) {
	go func() {
		handler.listeners[msg.DstListenerID] <- &ListenerNetMsg{
			FromPeerID:    from,
			FromHandlerID: msg.SrcHandlerID,
			Data:          msg.Data,
		}
	}()
}
