package net

import (
	"bytes"
	"encoding/gob"
	"net"

	"github.com/gcbb/src/common"
)

type NetMsgType uint8

const MAX_TTL = 65

const (
	MSG_PING  NetMsgType = iota
	MSG_PONG  NetMsgType = iota
	MSG_CONN  NetMsgType = iota
	MSG_APPLI NetMsgType = iota
)

type NetMsg struct {
	Src  common.NodeID
	Dst  common.NodeID
	Type NetMsgType
	Data []byte
	TTL  uint32
}

type NetResult struct {
	Id   common.NodeID
	Data []byte
	Addr *net.UDPAddr
}

type NetEncoder interface {
	Encode(interface{}) []byte
	Decode([]byte, interface{})
}

type GobNetEncoder struct{}

func (gne *GobNetEncoder) Encode(val interface{}) []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	enc.Encode(val)
	return buf.Bytes()
}
func (gne *GobNetEncoder) Decode(data []byte, val interface{}) {
	decoder := gob.NewDecoder(bytes.NewReader(data))
	decoder.Decode(val)
}

//TODO
func GetSelfPubIp() net.IP {
	return net.IP{}
}

type NetHandler interface {
	Send(common.NodeID, []byte)
	AddListener(string, chan *[]byte)
	RemoveListener(string)
	GetPeers(uint32) []common.NodeID
	Run()
}

type NaiveNetHandler struct {
	listeners map[string]chan *[]byte
}

func NewNaiveNetHandler() *NaiveNetHandler {
	ret := &NaiveNetHandler{listeners: make(map[string]chan *[]byte)}
	return ret
}

func (nnh *NaiveNetHandler) Send(peer common.NodeID, data []byte) {

}

func (nnh *NaiveNetHandler) AddListener(lid string, listener chan *[]byte) {
	nnh.listeners[lid] = listener
}

func (nnh *NaiveNetHandler) RemoveListener(lid string) {
	delete(nnh.listeners, lid)
}

func Run() {
	for {
	}
}
