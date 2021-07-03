package net

import (
	"time"

	"github.com/gcbb/src/common"
)

const StaticHandlerID uint16 = 0

type AppliNetMsg struct {
	HandlerID  uint16
	ListenerID uint16
	Data       []byte
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
	DstListenerID uint16
}

type AppliNetHandler interface {
	GetID() uint32
	AddListener(listenerID common.AppliListenerID, listenerChan chan *ListenerNetMsg)
	RemoveListener(listenerID common.AppliListenerID)
	SendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte)
	ReliableSendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte, id uint32, resultChan chan *SendResult)
	EstimateTimeOut(byteCnt uint32) time.Duration
}

type NaiveAppliNetHandler struct{}

func (handler *NaiveAppliNetHandler) GetID() uint32 { return 0 }
func (handler *NaiveAppliNetHandler) AddListener(listenerID common.AppliListenerID, listenerChan chan *ListenerNetMsg) {
}
func (handler *NaiveAppliNetHandler) RemoveListener(listenerID common.AppliListenerID) {}
func (handler *NaiveAppliNetHandler) SendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte) {
}
func (handler *NaiveAppliNetHandler) ReliableSendTo(peer common.NodeID, handlerID uint16, listenerID common.AppliListenerID, data []byte, id uint32, resultChan chan *SendResult) {
}
func (handler *NaiveAppliNetHandler) EstimateTimeOut(byteCnt uint32) time.Duration {
	return time.Duration(byteCnt) * time.Second
}
