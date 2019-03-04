package main

import (
	"fmt"
	"net/http"
)

func main() {

	var (
		userRepo UserRepository
		linkRepo LinkRepository
		voteRepo VoteRepository
		testRepo TestRepository
		radio    Radio
		service  *ServiceImpl
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

	radio = Radio{}
	fmt.Println(radio)

	httpRouter := NewHTTPRouter(service)
	http.ListenAndServe(":3000", httpRouter)
}
