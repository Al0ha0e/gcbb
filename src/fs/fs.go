package fs

import "sync"

type FS interface {
	Set(string, []byte)
	Get(string) ([]byte, error)
	Share(string, chan interface{})
	Purchase(string, chan interface{})
	Run()
}

type NaiveFS struct {
	db               sync.Map
	sessionId        uint64
	shareSessions    []*ShareSession
	purchaseSessions []*PurchaseSession
}

func NewNaiveFS() *NaiveFS {
	ret := &NaiveFS{}
	return ret
}

func (nfs *NaiveFS) Set(k string, v []byte) {
	nfs.db.Store(k, v)
}

func (nfs *NaiveFS) Get(k string) ([]byte, error) {
	ret, _ := nfs.db.Load(k)
	return ret.([]byte), nil
}

func (nfs *NaiveFS) Share(k string) {

}

func (nfs *NaiveFS) Purchase(url string) {}

func (nfs *NaiveFS) Run() {

}
