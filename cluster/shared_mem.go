package cluster

import (
	"sync"
)

type SharedMem struct {
	Shm     map[string]interface{}
	ShmLock *sync.Mutex

	// UpdateChan notifies the receiver that some update has happened
	// whicn can be trasmitted to concerned nodes
	// ONLY FOR MASTER
	UpdateChan chan interface{}
}

func NewSharedMem() *SharedMem {
	return &SharedMem{
		Shm:        make(map[string]interface{}),
		ShmLock:    &sync.Mutex{},
		UpdateChan: make(chan interface{}, 5),
	}
}

func (this *SharedMem) WriteVar(varname string, value interface{}) {
	this.ShmLock.Lock()
	this.Shm[varname] = value
	this.ShmLock.Unlock()
	this.UpdateChan <- true
}

func (this *SharedMem) Update(newmem map[string]interface{}) {
	this.ShmLock.Lock()
	for k := range this.Shm {
		delete(this.Shm, k)
	}
	for varname, value := range newmem {
		this.Shm[varname] = value
	}
	this.ShmLock.Unlock()
}

func (this *SharedMem) ReadVar(varname string) interface{} {
	if value, exists := this.Shm[varname]; exists {
		return value
	}
	return nil
}
