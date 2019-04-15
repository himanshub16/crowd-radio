// this file deals with the global state of the system
package main

import (
	"fmt"
	"github.com/google/uuid"
	"sort"
	"sync"
	"time"
)

type Radio struct {
	queue                []Link
	nowPlaying           *Link
	playerStartTimeSec   uint64
	playerCurTimeSec     uint64
	ticker               *time.Ticker
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
}

type HookType string

const (
	nowPlayingHook HookType = "nowPlaying"
	playerTimeHook HookType = "playerTime"
	queueHook      HookType = "queue"
)

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

func NewRadio(__service Service) *Radio {
	_service = __service
	return &Radio{
		nowPlaying:           nil,
		playerCurTimeSec:     0,
		playerStartTimeSec:   0,
		queue:                make([]Link, 0),
		tickResSec:           1,
		queueRefreshDur:      time.Second * 2,
		queueCapacity:        5,
		nowPlayingHooks:      make(map[uuid.UUID](chan interface{})),
		nowPlayingHooksMutex: &sync.Mutex{},
		playerTimeHooks:      make(map[uuid.UUID](chan interface{})),
		playerTimeHooksMutex: &sync.Mutex{},
		queueHooks:           make(map[uuid.UUID](chan interface{})),
		queueHooksMutex:      &sync.Mutex{},
	}
}

func (r *Radio) Engine() {
	go func() {
		for t := range r.ticker.C {
			if len(r.queue) == 0 ||
				t.After(r.nextQueueRefreshAt) {
				gotSome := r.refreshQueue()
				if gotSome == 0 {
					fmt.Println("There are no links available.")
					continue
				}
			}
			if r.nowPlaying == nil ||
				r.playerCurTimeSec > uint64(r.nowPlaying.Duration) {
				// first set the current song as expired
				r.nowPlaying = &r.queue[0]
				r.playerStartTimeSec = uint64(t.Unix())
				r.queue = r.queue[1:len(r.queue)]

				r.nowPlaying.IsExpired = true
				_service.UpdateLink(*r.nowPlaying)

				r.broadcastUpdate(nowPlayingHook)
				fmt.Println("now playing changed to", r.nowPlaying.LinkID)

			}
			r.playerCurTimeSec = uint64(t.Unix()) - r.playerStartTimeSec
			r.broadcastUpdate(playerTimeHook)

			r.ReorderQueue()
			r.broadcastUpdate(queueHook)
			// r.curState.NowPlaying = *r.nowPlaying
			// r.curState.PlayerCurTimeSec = r.playerCurTimeSec
			// r.curState.Queue = r.queue
			// r.broadcastUpdate()

			fmt.Println(t.Unix(), r.nowPlaying.LinkID, r.playerCurTimeSec)
		}
		fmt.Println("engine stopped")
	}()
}

func (r *Radio) Start() {
	// start an asynchronous radio which manages player state with time
	r.nowPlaying = nil
	r.playerCurTimeSec = 0
	r.playerStartTimeSec = 0
	r.ticker = time.NewTicker(time.Second)
	r.Engine()
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
	// close and perform cleanup if required
	r.ticker.Stop()

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

func (r *Radio) broadcastUpdate(htype HookType) {
	switch htype {
	case nowPlayingHook:
		for id := range r.nowPlayingHooks {
			// already a struct / can be marshalled to json
			r.nowPlayingHooks[id] <- *r.nowPlaying
		}
	case queueHook:
		for id := range r.queueHooks {
			// already a struct / can be marshalled to json
			r.queueHooks[id] <- r.queue
		}
	case playerTimeHook:
		for id := range r.playerTimeHooks {
			// already a struct / can be marshalled to json
			r.playerTimeHooks[id] <- r.playerCurTimeSec
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
