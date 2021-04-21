package fs

type ShareSessionState uint8

const (
	SHARE_ST      ShareSessionState = iota
	SHARE_SENDING ShareSessionState = iota
	SHARE_FIN     ShareSessionState = iota
)

type ShareSession struct {
	Id        uint64
	Peers     []int64
	PeerState []ShareSessionState
}

func NewShareSession() *ShareSession {
	return &ShareSession{}
}

type PurchaseSession struct {
	Id uint64
}

func NewPurchaseSession() *PurchaseSession {
	return &PurchaseSession{}
}
