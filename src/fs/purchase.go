package fs

type PurchaseSession struct {
	Id uint64
}

func NewPurchaseSession() *PurchaseSession {
	return &PurchaseSession{}
}
