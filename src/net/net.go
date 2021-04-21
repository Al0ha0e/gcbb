package net

type NetHandler interface {
	Send(int64, []byte)
	AddListener(string, chan *[]byte)
	RemoveListener(string)
	GetPeers(uint32) []int64
	Run()
}

type NaiveNetHandler struct {
}
