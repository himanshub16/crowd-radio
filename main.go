package main

import (
	"sync"
)

func main() {

	var (
		userRepo UserRepository
		linkRepo LinkRepository
		voteRepo VoteRepository
		testRepo TestRepository
		radio    *Radio
		service  *ServiceImpl
		wg       sync.WaitGroup
	)

	// just sqlite3 for now
	sqlitedb := NewSQLiteRepository("db.sqlite3")
	userRepo = sqlitedb
	linkRepo = sqlitedb
	voteRepo = sqlitedb
	testRepo = sqlitedb

	service = &ServiceImpl{
		userRepo: userRepo,
		linkRepo: linkRepo,
		voteRepo: voteRepo,
		testRepo: testRepo,
	}
	defer service.close()

	radio = NewRadio(service)
	wg.Add(1)
	go radio.Start()
	defer func() {
		radio.Shutdown()
		wg.Add(-1)
	}()

	echoRouter := NewHTTPRouter(service, radio)
	echoRouter.Start(":3000")
	wg.Wait()
}
