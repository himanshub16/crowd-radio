package main

// TODO implement peer-to-peer leader election somehow

import (
	"context"
	"flag"
	"github.com/google/uuid"
	"github.com/himanshub16/upnext-backend/cluster"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
)

var (
	runDs      bool
	apiUrl     string
	discoUrl   string
	clusterUrl string
	nodeID     string
	authToken  string
	wg         sync.WaitGroup
)

func parseFlags() {
	flag.BoolVar(&runDs, "runds", false, "Run discovery service")
	flag.StringVar(&apiUrl, "apiurl", "127.0.0.1:3000", "URL for ReST API")
	flag.StringVar(&discoUrl, "discourl", "127.0.0.1:4000", "URL for cluster discovery")
	flag.StringVar(&clusterUrl, "clusterurl", "ws://127.0.0.1:5000", "URL for cluster service to start")
	flag.StringVar(&authToken, "authtoken", "secrettoken", "Auth token for cluster nodes")

	u, _ := uuid.NewUUID()
	nodeID = u.String()

	flag.Parse()
}

func prepareWebService() *ServiceImpl {
	var (
		userRepo UserRepository
		linkRepo LinkRepository
		voteRepo VoteRepository
		testRepo TestRepository

		pgdb     *PostgresRepository
		sqlitedb *SQLiteRepository
	)
	var dbUrl string = os.Getenv("DB_URL")

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
	service := &ServiceImpl{
		userRepo: userRepo,
		linkRepo: linkRepo,
		voteRepo: voteRepo,
		testRepo: testRepo,
	}
	return service
}

func main() {
	parseFlags()
	rand.Seed(time.Now().UnixNano())

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	me := cluster.NodeInfoT{
		NodeID:   nodeID,
		URL:      clusterUrl,
		Priority: rand.Intn(100),
	}
	log.Println("starting cluster at ", clusterUrl)
	log.Println("This is me : ", me)

	service := prepareWebService()
	c := cluster.NewClusterService(clusterUrl, discoUrl, me, authToken)
	r := NewRadio(service, c.Shm)
	log.Println(r.shm, r.nowPlaying)
	apiRouter := NewHTTPRouter(service, r)

	go c.Start()
	for {
		select {
		case <-interrupt:
			c.Shutdown()
			r.Shutdown()
			apiRouter.Shutdown(context.Background())
			log.Println("stopping api router")
		case isLeader := <-c.SwitchMode:
			if isLeader {
				r.SwitchMode(masterRadio)
				apiRouter.Shutdown(context.Background())
				log.Println("stopping api router")
			} else {
				r.SwitchMode(peerRadio)
				go apiRouter.Start(apiUrl)
				log.Println("starting http router")
			}
		}
	}

	log.Println("time ends now")
}
