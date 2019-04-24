// this file deals with the global state of the system
package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/himanshub16/upnext-backend/cluster"
	"sort"
	"sync"
	"time"
)

type RadioType string
type HookType string

const (
	masterRadio RadioType = "masterRadio"
	peerRadio   RadioType = "peerRadio"
)
const (
	nowPlayingHook HookType = "nowPlaying"
	playerTimeHook HookType = "playerTime"
	queueHook      HookType = "queue"
)

type Radio struct {
	radioType RadioType
	shm       *cluster.SharedMem
	running   bool

	queue                []Link
	nowPlaying           *Link
	playerStartTimeSec   uint64
	playerCurTimeSec     uint64
	tickResSec           time.Duration
	queueRefreshDur      time.Duration
	nextQueueRefreshAt   time.Time
	queueCapacity        int64
	nowPlayingHooks      map[uuid.UUID](chan interface{})
	nowPlayingHooksMutex *sync.Mutex
	playerTimeHooks      map[uuid.UUID](chan interface{})
	playerTimeHooksMutex *sync.Mutex
	queueHooks           map[uuid.UUID](chan interface{})
	queueHooksMutex      *sync.Mutex

	interrupt chan interface{}
}

func IsValidHookType(htype HookType) bool {
	switch htype {
	case nowPlayingHook:
		return true
	case playerTimeHook:
		return true
	case queueHook:
		return true
	default:
		return false
	}
}

// type RadioState struct {
// 	NowPlaying       Link   `json:"now_playing"`
// 	PlayerCurTimeSec uint64 `json:"player_cur_time_sec"`
// 	Queue            []Link `json:"queue"`
// }

var _service Service

func NewRadio(__service Service, shm *cluster.SharedMem) *Radio {
	_service = __service
	return &Radio{
		shm:     shm,
		running: false,

		nowPlaying:           nil,
		playerCurTimeSec:     0,
		playerStartTimeSec:   0,
		queue:                make([]Link, 0),
		tickResSec:           1,
		queueRefreshDur:      time.Second * 10,
		queueCapacity:        5,
		nowPlayingHooks:      make(map[uuid.UUID](chan interface{})),
		nowPlayingHooksMutex: &sync.Mutex{},
		playerTimeHooks:      make(map[uuid.UUID](chan interface{})),
		playerTimeHooksMutex: &sync.Mutex{},
		queueHooks:           make(map[uuid.UUID](chan interface{})),
		queueHooksMutex:      &sync.Mutex{},

		interrupt: make(chan interface{}, 1),
	}
}

func (r *Radio) MasterEngine() {
	r.nowPlaying = nil
	r.playerCurTimeSec = 0
	r.playerStartTimeSec = 0

	ticker := time.NewTicker(time.Second * r.tickResSec)
	defer ticker.Stop()

	for {
		select {
		case <-r.interrupt:
			return
		case t := <-ticker.C:
			r.singleIteration(t)
		}
	}
}

func (r *Radio) PeerEngine() {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			if t.After(r.shm.LastUpdatedAt) {
				r.updateStateFromShm()
				fmt.Println("shm updated", r.nowPlaying, r.queue, r.playerCurTimeSec)
				r.broadcastUpdate(nowPlayingHook, r.nowPlaying)
				r.broadcastUpdate(queueHook, r.queue)
				r.broadcastUpdate(playerTimeHook, r.playerCurTimeSec)
			}
		case <-r.interrupt:
			return
			// case v := <-r.shm.PeerChan:
			// 	fmt.Println("radio got update to", v.Ts)
			// 	// r.broadcastUpdate(v.Ts, v.Mem)
			// 	// update radio parameters from hook variables
			// 	fmt.Println(r.shm.ReadVar(string(nowPlayingHook)))
		}
	}
}

func (r *Radio) updateStateFromShm() {
	// nowPlaying
	np, err := json.Marshal(r.shm.ReadVar(string(nowPlayingHook)))
	if err != nil {
		fmt.Println("failed to marshal nowplaying")
	}
	if err = json.Unmarshal(np, &r.nowPlaying); err != nil {
		fmt.Println("failed to unmarshal nowplaying")
	}

	pt, err := json.Marshal(r.shm.ReadVar(string(playerTimeHook)))
	if err != nil {
		fmt.Println("failed to marshal nowplaying")
	}
	if err = json.Unmarshal(pt, &r.playerCurTimeSec); err != nil {
		fmt.Println("failed to unmarshal playerCurTimeSec")
	}

	q, err := json.Marshal(r.shm.ReadVar(string(queueHook)))
	if err != nil {
		fmt.Println("failed to marshal nowplaying")
	}
	if err = json.Unmarshal(q, &r.queue); err != nil {
		fmt.Println("failed to unmarshal queue")
	}
}

func (r *Radio) singleIteration(t time.Time) {

	if len(r.queue) == 0 ||
		t.After(r.nextQueueRefreshAt) {
		gotSome := r.refreshQueue()
		if gotSome == 0 {
			fmt.Println("There are no links available.")
		}
	} else if r.nowPlaying == nil ||
		r.playerCurTimeSec > uint64(r.nowPlaying.Duration) {
		// first set the current song as expired
		r.nowPlaying = &r.queue[0]
		r.playerStartTimeSec = uint64(t.Unix())
		r.queue = r.queue[1:len(r.queue)]

		r.nowPlaying.IsExpired = true
		_service.UpdateLink(*r.nowPlaying)

		// r.broadcastUpdate(nowPlayingHook, *r.nowPlaying)
		r.shm.WriteVar(string(nowPlayingHook), *r.nowPlaying, true)
		fmt.Println("now playing changed to", r.nowPlaying.LinkID)

		r.ReorderQueue()
		// r.broadcastUpdate(queueHook, r.queue)
		r.shm.WriteVar(string(queueHook), r.queue, true)

		// r.curState.NowPlaying = *r.nowPlaying
		// r.curState.PlayerCurTimeSec = r.playerCurTimeSec
		// r.curState.Queue = r.queue
		// r.broadcastUpdate()

	}

	if r.nowPlaying != nil {
		r.playerCurTimeSec = uint64(t.Unix()) - r.playerStartTimeSec
		// r.broadcastUpdate(playerTimeHook, r.playerCurTimeSec)
		r.shm.WriteVar(string(playerTimeHook), r.playerCurTimeSec, true)
		fmt.Println(t.Unix(), r.nowPlaying.LinkID, r.playerCurTimeSec)

		if r.playerCurTimeSec > uint64(r.nowPlaying.Duration) {
			r.nowPlaying = nil
		}
	} else {
		r.playerCurTimeSec = 1 << 30
		// r.broadcastUpdate(playerTimeHook, r.playerCurTimeSec)
		r.shm.WriteVar(string(playerTimeHook), r.playerCurTimeSec, true)
	}
}

func (r *Radio) Start() {
	if r.radioType == masterRadio {
		// start an asynchronous radio which manages player state with time
		r.MasterEngine()
	} else if r.radioType == peerRadio {
		// just subscribe to whatever channel is available
		r.PeerEngine()
	}
}

func (r *Radio) SwitchMode(newMode RadioType) {
	if r.running {
		r.Shutdown()
	}
	r.radioType = newMode
	go r.Start()
	r.running = true
}

func (r *Radio) refreshQueue() int {
	r.queue = _service.GetAllLinks(5)
	r.nextQueueRefreshAt = time.Now().Add(r.queueRefreshDur)
	return len(r.queue)
}

func (r *Radio) ReorderQueue() {
	// update total votes for all links in queue

	linkIDs := make([]int64, len(r.queue))
	for i, l := range r.queue {
		linkIDs[i] = l.LinkID
	}

	totalVotes := service.GetTotalVoteForLinks(linkIDs)

	for i, l := range r.queue {
		if score, ok := totalVotes[l.LinkID]; ok {
			r.queue[i].TotalVotes = score
		} else {
			r.queue[i].TotalVotes = 0
		}
	}

	// sort on the basis of total votes
	sort.Slice(r.queue, func(i, j int) bool {
		return r.queue[i].TotalVotes > r.queue[j].TotalVotes
	})
}

func (r *Radio) Shutdown() {
	// close engine
	r.interrupt <- true

	// clean and close all channels
	for id := range r.nowPlayingHooks {
		close(r.nowPlayingHooks[id])
	}
	for id := range r.queueHooks {
		close(r.queueHooks[id])
	}
	for id := range r.playerTimeHooks {
		close(r.playerTimeHooks[id])
	}
}

func (r *Radio) broadcastUpdate(htype HookType, msg interface{}) {
	fmt.Println("update to broadcast ", htype)
	switch htype {
	case nowPlayingHook:
		for id := range r.nowPlayingHooks {
			// already a struct / can be marshalled to json
			r.nowPlayingHooks[id] <- msg
		}
	case queueHook:
		for id := range r.queueHooks {
			// already a struct / can be marshalled to json
			r.queueHooks[id] <- msg
		}
	case playerTimeHook:
		for id := range r.playerTimeHooks {
			// already a struct / can be marshalled to json
			r.playerTimeHooks[id] <- msg
		}
	}
}

func (r *Radio) RegisterHook(htype HookType) (uuid.UUID, chan interface{}) {
	id := uuid.New()
	c := make(chan interface{})

	switch htype {

	case nowPlayingHook:
		r.nowPlayingHooksMutex.Lock()
		r.nowPlayingHooks[id] = c
		defer r.nowPlayingHooksMutex.Unlock()

	case queueHook:
		r.queueHooksMutex.Lock()
		r.queueHooks[id] = c
		defer r.queueHooksMutex.Unlock()

	case playerTimeHook:
		r.playerTimeHooksMutex.Lock()
		r.playerTimeHooks[id] = c
		defer r.playerTimeHooksMutex.Unlock()
	}

	return id, c
}

func (r *Radio) DeregisterHook(htype HookType, id uuid.UUID) {
	switch htype {

	case nowPlayingHook:
		r.nowPlayingHooksMutex.Lock()
		delete(r.nowPlayingHooks, id)
		defer r.nowPlayingHooksMutex.Unlock()

	case queueHook:
		r.queueHooksMutex.Lock()
		delete(r.queueHooks, id)
		defer r.queueHooksMutex.Unlock()

	case playerTimeHook:
		r.playerTimeHooksMutex.Lock()
		delete(r.playerTimeHooks, id)
		defer r.playerTimeHooksMutex.Unlock()
	}
}
