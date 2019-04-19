package main

// TODO implement peer-to-peer leader election somehow

import (
	"log"
	"net/url"
	"os"
	"sync"
)

func main() {

	var (
		userRepo UserRepository
		linkRepo LinkRepository
		voteRepo VoteRepository
		testRepo TestRepository

		dbUrl    string
		pgdb     *PostgresRepository
		sqlitedb *SQLiteRepository

		radio   *Radio
		service *ServiceImpl
		wg      sync.WaitGroup
	)

	dbUrl = os.Getenv("DB_URL")
	log.Println("database url", dbUrl)
	if u, err := url.Parse(dbUrl); err == nil {
		switch u.Scheme {
		case "sqlite":
			sqlitedb = NewSQLiteRepository(u.Hostname())
			userRepo = sqlitedb
			linkRepo = sqlitedb
			voteRepo = sqlitedb
			testRepo = sqlitedb

		case "postgres":
			pgdb = NewPostgresRepository(dbUrl)
			userRepo = pgdb
			linkRepo = pgdb
			voteRepo = pgdb
			testRepo = pgdb
		}
	}

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
