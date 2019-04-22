package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type dsNodeT struct {
	Url    string `json:"url"`
	NodeID string `json:"node_id"`
}

type DiscoveryService struct {
	connectedNodes map[string]string
	joiningTime    map[string]time.Time
	connMutex      sync.Mutex
	authToken      string
}

func (_ *DiscoveryService) healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Discovery service is healthy and running!")
}

func (ds *DiscoveryService) joinHandler(w http.ResponseWriter, r *http.Request) {
	presentNodesB, _ := json.Marshal(ds.connectedNodes)
	presentNodesStr := string(presentNodesB)

	var curNode dsNodeT
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&curNode); err != nil {
		log.Println("joinHandler: decode failed", "\n", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Bad request")

		return
	}

	ds.connMutex.Lock()
	ds.connectedNodes[curNode.NodeID] = curNode.Url
	ds.joiningTime[curNode.NodeID] = time.Now().Add(time.Second * 2)
	ds.connMutex.Unlock()
	fmt.Fprintf(w, presentNodesStr)

	log.Println("added ", curNode.Url, " nodeID: ", curNode.NodeID)
}

func (ds *DiscoveryService) StartService(myurl string) {

	http.HandleFunc("/health", ds.healthHandler)
	http.HandleFunc("/join", ds.joinHandler)

	fmt.Println("starting at url", myurl)
	fmt.Println("auth token", ds.authToken)
	http.ListenAndServe(myurl, nil)
}

func (ds *DiscoveryService) performHealthCheck() {
	ticker := time.NewTicker(time.Second * 2)
	for t := range ticker.C {
		for nodeID, addr := range ds.connectedNodes {
			if t.Before(ds.joiningTime[nodeID]) {
				continue
			}

			addr = fmt.Sprint("http://", addr, "/health")
			req, err := http.NewRequest("GET", addr, nil)
			if err != nil {
				log.Fatal("failed to create request err:", err)
			}
			req.Header.Set("auth_token", ds.authToken)

			client := http.Client{}
			res, err := client.Do(req)

			if err != nil {
				ds.removeNode(nodeID, err)
				continue
			}
			if res.StatusCode != http.StatusOK {
				ds.removeNode(nodeID, nil)
				continue
			}
			res.Body.Close()
		}
	}
}

func (ds *DiscoveryService) removeNode(nodeID string, err error) {
	log.Println("removing node id:", nodeID, " err:", err)

	ds.connMutex.Lock()
	delete(ds.connectedNodes, nodeID)
	ds.connMutex.Unlock()
}

func main() {
	var url string
	var authToken string
	flag.StringVar(&url, "url", "127.0.0.1:9090", "Address of discovery service")
	flag.StringVar(&authToken, "authtoken", "secrettoken", "Auth token")
	flag.Parse()

	ds := DiscoveryService{
		connectedNodes: make(map[string]string),
		joiningTime:    make(map[string]time.Time),
		connMutex:      sync.Mutex{},
		authToken:      authToken,
	}

	go ds.performHealthCheck()
	ds.StartService(url)
}
