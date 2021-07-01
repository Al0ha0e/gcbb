package net

import "github.com/gcbb/src/common"

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

type AppliNetHandler interface {
	GetID() uint32
	AddListener(listenerID uint16, listenerChan chan ListenerNetMsg)
	RemoveListener(listenerID uint16)
	SendTo(peer common.NodeID, handlerID uint16, listenerID uint16, data []byte)
}
