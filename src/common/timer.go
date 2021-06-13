package common

import (
	"context"
	"time"
)

type Timer struct {
	id          uint32
	timer       *time.Timer
	duration    time.Duration
	ctrlContext context.Context
	cancelFunc  context.CancelFunc
	timeUpChan  chan uint32
}

func NewTimer(id uint32, parent context.Context, tuChan chan uint32) *Timer {
	ctx, fnc := context.WithCancel(parent)
	ret := &Timer{
		id:          id,
		timer:       time.NewTimer(1 * time.Second),
		ctrlContext: ctx,
		cancelFunc:  fnc,
		timeUpChan:  tuChan,
	}
	ret.timer.Stop()
	return ret
}

func (tm *Timer) Start(du time.Duration) {
	go tm.run(du)
}

func (tm *Timer) GetDuration() time.Duration {
	return tm.duration
}

func (tm *Timer) Stop() {
	tm.cancelFunc()
}

func (tm *Timer) run(du time.Duration) {
	tm.timer.Reset(du)
	for {
		select {
		case <-tm.ctrlContext.Done():
			tm.timer.Stop()
			return
		case <-tm.timer.C:
			tm.timeUpChan <- tm.id
		}
	}
}
