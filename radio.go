// this file deals with the global state of the system
package main

import (
	"fmt"
	"time"
)

type Radio struct {
	queue              []Link
	nowPlaying         *Link
	playerStartTimeSec uint64
	playerCurTimeSec   uint64
	ticker             *time.Ticker
	tickResSec         time.Duration
	queueRefreshDur    time.Duration
	nextQueueRefreshAt time.Time
}

var _service Service

func NewRadio(__service Service) *Radio {
	_service = __service
	fmt.Println(_service.GetAllLinks())
	return &Radio{
		nowPlaying:         nil,
		playerCurTimeSec:   0,
		playerStartTimeSec: 0,
		queue:              make([]Link, 0),
		tickResSec:         1,
		queueRefreshDur:    time.Minute * 1,
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
				// r.playerStartTimeSec+uint64(r.nowPlaying.Duration) > uint64(t.Unix()) {
				r.nowPlaying = &r.queue[0]
				fmt.Println("now playing changed to", r.nowPlaying.LinkID)
				r.playerStartTimeSec = uint64(t.Unix())
				r.queue = r.queue[1:len(r.queue)]
			}
			r.playerCurTimeSec = uint64(t.Unix()) - r.playerStartTimeSec
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
	r.queue = _service.GetAllLinks()
	r.nextQueueRefreshAt = time.Now().Add(r.queueRefreshDur)
	return len(r.queue)
}

func (r *Radio) Shutdown() {
	// close and perform cleanup if required
	r.ticker.Stop()
}
