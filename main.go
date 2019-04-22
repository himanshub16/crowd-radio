package main

// TODO implement peer-to-peer leader election somehow

import (
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

// TODO pass required variables here to do cleanup
func performCleanup() {
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	<-sigint

	// we have received an interrupt, cleanup is required
}

func main() {
	parseFlags()
	rand.Seed(time.Now().Unix())

	if runDs {
	} else {
		me := cluster.NodeInfoT{
			NodeID:   nodeID,
			URL:      clusterUrl,
			Priority: rand.Intn(100),
		}
		log.Println("starting cluster at ", clusterUrl)
		log.Println("This is me : ", me)
		c := cluster.NewClusterService(clusterUrl, discoUrl, me, authToken)
		c.Start()
		return

		service := prepareWebService()
		// if running in replicated mode
		// leader election would have happened
		// and we know if this is a master/slave

		radio = NewRadio(service)
		wg.Add(1)
		go radio.Start()
		defer func() {
			radio.Shutdown()
			wg.Add(-1)
		}()

		echoRouter := NewHTTPRouter(service, radio)
		echoRouter.Start(apiUrl)
	}

	wg.Wait()
}
