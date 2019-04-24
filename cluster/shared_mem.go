package cluster

import (
	"sync"
	"time"
)

type UpdateEvent struct {
	Ts  time.Time
	Mem interface{}
}

type SharedMem struct {
	Shm           map[string]interface{}
	LastUpdatedAt time.Time
	ShmLock       *sync.Mutex

	// UpdateChan notifies the receiver that some update has happened
	// whicn can be trasmitted to concerned nodes
	// ONLY FOR MASTER
	PeerChan   chan UpdateEvent
	MasterChan chan UpdateEvent
}

func NewSharedMem() *SharedMem {
	return &SharedMem{
		Shm:           make(map[string]interface{}),
		LastUpdatedAt: time.Now(),
		ShmLock:       &sync.Mutex{},
		MasterChan:    make(chan UpdateEvent, 5),
		PeerChan:      make(chan UpdateEvent, 5),
	}
}

func (this *SharedMem) WriteVar(varname string, value interface{}, isMaster bool) {
	this.ShmLock.Lock()
	this.Shm[varname] = value
	this.ShmLock.Unlock()

	evt := UpdateEvent{
		Ts:  time.Now(),
		Mem: this.Shm,
	}

	if isMaster {
		this.MasterChan <- evt
		// } else {
		// 	this.PeerChan <- evt
	}
}

func (this *SharedMem) Update(newmem map[string]interface{}) {
	this.ShmLock.Lock()
	for k := range this.Shm {
		delete(this.Shm, k)
	}
	for varname, value := range newmem {
		this.Shm[varname] = value
	}
	this.LastUpdatedAt = time.Now()
	this.ShmLock.Unlock()
}

func (this *SharedMem) ReadVar(varname string) interface{} {
	if value, exists := this.Shm[varname]; exists {
		return value
	}
	return nil
}
