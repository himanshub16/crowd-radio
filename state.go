// this file deals with the global state of the system
package main

type Radio struct {
	nowPlaying    *Link
	linkRepo      *LinkRepository
	playerTimeSec uint64
	leaderBoard   []*Link
}

func NewRadio() *Radio {
	return &Radio{
		nowPlaying:    nil,
		playerTimeSec: 0,
		leaderBoard:   make([]*Link, 0),
	}
}

func (r *Radio) Start() {
	// start an asynchronous radio which manages player state with time
}

func (r *Radio) Resync() {
	// resync state variables for next song to play
}

func (r *Radio) Shutdown() {
	// close and perform cleanup if required
}
