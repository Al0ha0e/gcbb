package net

type LoopedQueue struct {
	data    []interface{}
	queueSt uint32
	queueEn uint32
	size    uint32
	stSeq   uint32
	enSeq   uint32
}

func NewLoopedQueue(size uint32, seq uint32) *LoopedQueue {
	return &LoopedQueue{
		data:    make([]interface{}, size+1),
		queueSt: 0,
		queueEn: 0,
		stSeq:   seq,
		enSeq:   seq,
		size:    size,
	}
}

func (lq *LoopedQueue) Full() bool {
	return (lq.queueEn-lq.queueSt+lq.size+1)%(lq.size+1) == lq.size
}

func (lq *LoopedQueue) Length() uint32 {
	return (lq.queueEn - lq.queueSt + lq.size + 1) % (lq.size + 1)
}

func (lq *LoopedQueue) Size() uint32 {
	return lq.size
}

func (lq *LoopedQueue) IsValidSeq(seq uint32) bool {
	return seq-lq.stSeq < lq.Length()
}

func (lq *LoopedQueue) Get(id uint32) interface{} {
	if (id-lq.queueSt+lq.size+1)%(lq.size+1) < lq.Length() {
		return lq.data[id]
	}
	return nil
}

func (lq *LoopedQueue) Set(id uint32, data interface{}) {
	if (id-lq.queueSt+lq.size+1)%(lq.size+1) < lq.Length() {
		lq.data[id] = data
	}
}

func (lq *LoopedQueue) GetBySeq(seq uint32) interface{} {
	if lq.IsValidSeq(seq) {
		return lq.Get(seq - lq.stSeq)
	}
	return nil
}

func (lq *LoopedQueue) SetBySeq(seq uint32, data interface{}) {
	if lq.IsValidSeq(seq) {
		lq.Set(seq-lq.stSeq, data)
	}
}

func (lq *LoopedQueue) GetEnSeq() uint32 {
	return lq.enSeq
}

func (lq *LoopedQueue) GetStSeq() uint32 {
	return lq.stSeq
}

func (lq *LoopedQueue) PopFront() interface{} {
	if lq.queueSt == lq.queueEn {
		return nil
	}
	ret := lq.data[lq.queueSt]
	lq.queueSt = (lq.queueSt + 1) % (lq.size + 1)
	lq.stSeq += 1
	return ret
}

func (lq *LoopedQueue) PushBack(obj interface{}) bool {
	if (lq.queueEn-lq.queueSt+lq.size+1)%(lq.size+1) != lq.size {
		lq.data[lq.queueEn] = obj
		lq.queueEn = (lq.queueEn + 1) % (lq.size + 1)
		lq.enSeq += 1
		return true
	}
	return false
}
