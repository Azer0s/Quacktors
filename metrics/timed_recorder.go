package metrics

import (
	"go.uber.org/atomic"
	"time"
)

type TimedRecorderHook interface {
	RecordSpawn(count int32)
	RecordDie(count int32)
	RecordDrop(count int32)
	RecordDropRemote(count int32)
	RecordUnhandled(count int32)
	RecordReceiveTotal(count int32)
	RecordReceiveRemote(count int32)
	RecordSendLocal(count int32)
	RecordSendRemote(count int32)
}

func NewTimedRecorder(hook TimedRecorderHook, interval time.Duration) *TimedRecorder {
	return &TimedRecorder{
		hook:     hook,
		interval: interval,
	}
}

type TimedRecorder struct {
	spawnCount         *atomic.Int32
	dieCount           *atomic.Int32
	dropCount          *atomic.Int32
	remoteDropCount    *atomic.Int32
	unhandledCount     *atomic.Int32
	receiveTotalCount  *atomic.Int32
	receiveRemoteCount *atomic.Int32
	sendLocalCount     *atomic.Int32
	sendRemoteCount    *atomic.Int32
	hook               TimedRecorderHook
	interval           time.Duration
}

func (t *TimedRecorder) Init() {
	t.spawnCount = atomic.NewInt32(0)
	t.dieCount = atomic.NewInt32(0)
	t.dropCount = atomic.NewInt32(0)
	t.remoteDropCount = atomic.NewInt32(0)
	t.unhandledCount = atomic.NewInt32(0)
	t.receiveTotalCount = atomic.NewInt32(0)
	t.receiveRemoteCount = atomic.NewInt32(0)
	t.sendLocalCount = atomic.NewInt32(0)
	t.sendRemoteCount = atomic.NewInt32(0)

	go func() {
		for {
			<-time.After(t.interval)

			go t.hook.RecordSpawn(t.spawnCount.Swap(0))
			go t.hook.RecordDie(t.dieCount.Swap(0))
			go t.hook.RecordDrop(t.dropCount.Swap(0))
			go t.hook.RecordDropRemote(t.remoteDropCount.Swap(0))
			go t.hook.RecordUnhandled(t.unhandledCount.Swap(0))
			go t.hook.RecordReceiveTotal(t.receiveTotalCount.Swap(0))
			go t.hook.RecordReceiveRemote(t.receiveRemoteCount.Swap(0))
			go t.hook.RecordSendLocal(t.sendLocalCount.Swap(0))
			go t.hook.RecordSendRemote(t.sendRemoteCount.Swap(0))
		}
	}()
}

func (t *TimedRecorder) RecordSpawn(pid string) {
	t.spawnCount.Inc()
}

func (t *TimedRecorder) RecordDie(pid string) {
	t.dieCount.Inc()
}

func (t *TimedRecorder) RecordDrop(pid string, amount int) {
	t.dropCount.Add(int32(amount))
}

func (t *TimedRecorder) RecordDropRemote(machineId string, amount int) {
	t.remoteDropCount.Add(int32(amount))
}

func (t *TimedRecorder) RecordUnhandled(target string) {
	t.unhandledCount.Inc()
}

func (t *TimedRecorder) RecordReceive(pid string) {
	t.receiveTotalCount.Inc()
}

func (t *TimedRecorder) RecordReceiveRemote(pid string) {
	t.receiveRemoteCount.Inc()
}

func (t *TimedRecorder) RecordSendLocal(target string) {
	t.sendLocalCount.Inc()
}

func (t *TimedRecorder) RecordSendRemote(target string) {
	t.sendRemoteCount.Inc()
}
