package net

import (
	"fmt"
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
	id         uint16
	listeners  map[common.AppliListenerID]chan *ListenerNetMsg
	netHandler NetHandler
}

func NewNaiveAppliNetHandler(id uint16, netHandler NetHandler) *NaiveAppliNetHandler {
	ret := &NaiveAppliNetHandler{
		id:         id,
		listeners:  make(map[common.AppliListenerID]chan *ListenerNetMsg),
		netHandler: netHandler,
	}
	ret.netHandler.AddAppliHandler(ret)
	return ret
}

func (handler *NaiveAppliNetHandler) GetID() uint16 { return handler.id }
func (handler *NaiveAppliNetHandler) AddListener(listenerID common.AppliListenerID, listenerChan chan *ListenerNetMsg) {
	handler.listeners[listenerID] = listenerChan
}
func (handler *NaiveAppliNetHandler) RemoveListener(listenerID common.AppliListenerID) {
	delete(handler.listeners, listenerID)
}

func (handler *NaiveAppliNetHandler) SendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte) {
	msg := NewAppliNetMsg(handler.id, handlerID, listenerID, data)
	handler.netHandler.SendTo(peer, msg)
}
func (handler *NaiveAppliNetHandler) ReliableSendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte, id uint32, resultChan chan *SendResult) {
	msg := NewAppliNetMsg(handler.id, handlerID, listenerID, data)
	handler.netHandler.ReliableSendTo(peer, msg, id, resultChan)
}
func (handler *NaiveAppliNetHandler) EstimateTimeOut(byteCnt uint32) time.Duration {
	return time.Duration(byteCnt) * time.Second
}

func (handler *NaiveAppliNetHandler) OnMsgArrive(from common.NodeID, msg *AppliNetMsg) {
	fmt.Println("ON MSG ARRIVE", from, msg)
	go func() {
		handler.listeners[msg.DstListenerID] <- &ListenerNetMsg{
			FromPeerID:    from,
			FromHandlerID: msg.SrcHandlerID,
			Data:          msg.Data,
		}
		fmt.Println("OK")
	}()
}

type AppliNetHandlerFactory interface {
	GetHandler() AppliNetHandler
}

type NaiveAppliNetHandlerFactory struct {
	id         uint16
	netHandler NetHandler
}

func NewNaiveAppliNetHandlerFactory(netHandler NetHandler) *NaiveAppliNetHandlerFactory {
	return &NaiveAppliNetHandlerFactory{
		id:         0,
		netHandler: netHandler,
	}
}

func (factory *NaiveAppliNetHandlerFactory) GetHandler() AppliNetHandler {
	factory.id += 1
	return NewNaiveAppliNetHandler(factory.id, factory.netHandler)
}
