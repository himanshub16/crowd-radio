// this file deals with the global state of the system
package main

import (
	"fmt"
	"sort"
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
	queueCapacity      int64
}

var _service Service

func NewRadio(__service Service) *Radio {
	_service = __service
	return &Radio{
		nowPlaying:         nil,
		playerCurTimeSec:   0,
		playerStartTimeSec: 0,
		queue:              make([]Link, 0),
		tickResSec:         1,
		queueRefreshDur:    time.Minute * 1,
		queueCapacity:      5,
	}
}

func (r *Radio) Engine() {
	go func() {
		for t := range r.ticker.C {
			if len(r.queue) < int(r.queueCapacity) ||
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

				fmt.Println("now playing changed to", r.nowPlaying.LinkID)

			}
			r.playerCurTimeSec = uint64(t.Unix()) - r.playerStartTimeSec

			r.ReorderQueue()

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
}
