package quacktors

import (
	"fmt"
	"go.uber.org/zap"
	"sync"
)

type Pid struct {
	MachineId       string
	Id              string
	quitChan        chan<- bool
	quitChanMu      *sync.RWMutex
	messageChan     chan<- Message
	messageChanMu   *sync.RWMutex
	monitorChan     chan<- *Pid
	monitorChanMu   *sync.RWMutex
	demonitorChan   chan<- *Pid
	demonitorChanMu *sync.RWMutex
	//Stores channels to scheduled tasks (monitors, SendAfter, monitors the actor itself launches but doesn't consume)
	scheduled map[string]chan bool
	//Stores channels to tell a monitor taks to quit (when a pid is demonitored)
	monitorQuitChannels map[string]chan bool
	//Is locked when `scheduled` or `monitorQuitChannels` is modified
	monitorSetupMu *sync.Mutex
}

func createPid(quitChan chan<- bool, messageChan chan<- Message, monitorChan chan<- *Pid, demonitorChan chan<- *Pid, scheduled map[string]chan bool, monitorQuitChannels map[string]chan bool) *Pid {
	pid := &Pid{
		MachineId:           machineId,
		Id:                  "",
		quitChan:            quitChan,
		quitChanMu:          &sync.RWMutex{},
		messageChan:         messageChan,
		messageChanMu:       &sync.RWMutex{},
		monitorChan:         monitorChan,
		monitorChanMu:       &sync.RWMutex{},
		demonitorChan:       demonitorChan,
		demonitorChanMu:     &sync.RWMutex{},
		scheduled:           scheduled,
		monitorQuitChannels: monitorQuitChannels,
		monitorSetupMu:      &sync.Mutex{},
	}

	registerPid(pid)

	return pid
}

func (pid *Pid) cleanup() {
	logger.Info("cleaning up pid", zap.String("pid_id", pid.Id))

	deletePid(pid.Id)

	pid.quitChanMu.Lock()
	close(pid.quitChan)
	pid.quitChan = nil
	pid.quitChanMu.Unlock()

	pid.messageChanMu.Lock()
	close(pid.messageChan)
	pid.messageChan = nil
	pid.messageChanMu.Unlock()

	pid.monitorChanMu.Lock()
	close(pid.monitorChan)
	pid.monitorChan = nil
	pid.monitorChanMu.Unlock()

	pid.demonitorChanMu.Lock()
	close(pid.demonitorChan)
	pid.demonitorChan = nil
	pid.demonitorChanMu.Unlock()

	//Terminate all scheduled events/send down message to monitor tasks
	pid.monitorSetupMu.Lock()
	logger.Debug("sending out scheduled events after pid cleanup", zap.String("pid_id", pid.Id))
	for n, ch := range pid.scheduled {
		ch <- true //this is blocking
		close(ch)
		delete(pid.scheduled, n)
	}

	//Delete monitorQuitChannels
	logger.Debug("deleting monitor abort channels after pid cleanup", zap.String("pid_id", pid.Id))
	for n, c := range pid.monitorQuitChannels {
		close(c)
		delete(pid.monitorQuitChannels, n)
	}
	pid.monitorSetupMu.Unlock()

	pid.monitorQuitChannels = nil
}

func (pid *Pid) setupMonitor(monitor *Pid) {
	logger.Info("setting up monitor",
		zap.String("monitored_pid", pid.String()),
		zap.String("monitor_pid", monitor.String()),
	)

	pid.monitorSetupMu.Lock()
	defer pid.monitorSetupMu.Unlock()

	monitorChannel := make(chan bool)
	pid.scheduled[monitor.String()] = monitorChannel

	monitorQuitChannel := make(chan bool)
	pid.monitorQuitChannels[monitor.String()] = monitorQuitChannel

	go func() {
		select {
		case <-monitorQuitChannel:
			return
		case <-monitorChannel:
			doSend(monitor, &DownMessage{Who: pid})
		}
	}()
}

func (pid *Pid) removeMonitor(monitor *Pid) {
	pid.monitorSetupMu.Lock()
	defer pid.monitorSetupMu.Unlock()

	name := monitor.String()

	pid.monitorQuitChannels[name] <- true

	close(pid.monitorQuitChannels[name])
	close(pid.scheduled[name])

	delete(pid.monitorQuitChannels, name)
	delete(pid.scheduled, name)

	logger.Info("monitor removed successfully",
		zap.String("monitored_pid", pid.String()),
		zap.String("monitor_pid", monitor.String()),
	)
}

func (pid *Pid) String() string {
	return fmt.Sprintf("%s@%s", pid.Id, pid.MachineId)
}
