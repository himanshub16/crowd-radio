package cluster

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type ClusterService struct {
	clusterUrl   string
	discoveryUrl string
	meshNet      *MeshNetwork

	broadcastChan chan Message
	// decisions based on current state
	lastBulliedAt          time.Time
	idleTimeToBecomeLeader time.Duration
	biggestBullySoFar      int
	IsLeader               bool
	leaderElected          bool

	SwitchMode chan bool

	Shm       *SharedMem
	interrupt chan interface{}
}

func NewClusterService(clusterUrl, discoveryUrl string, me NodeInfoT, authToken string) *ClusterService {
	return &ClusterService{
		clusterUrl:   clusterUrl,
		discoveryUrl: discoveryUrl,
		meshNet:      NewMeshNetwork(me, authToken),

		broadcastChan: make(chan Message, 5),

		lastBulliedAt:          time.Now(),
		idleTimeToBecomeLeader: time.Second * 5,
		biggestBullySoFar:      me.Priority,
		IsLeader:               false,
		leaderElected:          false,

		SwitchMode: make(chan bool),

		Shm:       NewSharedMem(),
		interrupt: make(chan interface{}, 1),
	}
}

func (this *ClusterService) manageIncomingMessages(parentwg *sync.WaitGroup) {
	go func() {
		defer parentwg.Done()
		this.handleBroadcasts()
	}()

	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			if t.After(this.lastBulliedAt.Add(this.idleTimeToBecomeLeader)) &&
				!this.leaderElected {
				if !this.IsLeader &&
					this.meshNet.me.Priority >= this.biggestBullySoFar {
					// the second condition makes sure the if part comes are at the required time

					this.IsLeader = true
					this.SwitchMode <- this.IsLeader
					log.Println("I proclaim myself as a leader.", this.IsLeader)

				} else {
					// let's wait for the right time to come, or someone is already the leader
					this.leaderElected = true
					this.SwitchMode <- this.IsLeader
					log.Println("Someone else is perhaps the leader.", this.IsLeader)
				}

				this.leaderElected = true
			}

		case nodeID := <-this.meshNet.soldierDown:
			this.handleSoldierDown(nodeID)

		case msg := <-this.meshNet.commonIncomingChan:

			switch msg.MsgType {
			case bullyMsg:
				this.handleBullyMsg(msg)

			case shmMsg:
				// our implementations only send writes to shared memory
				evt := msg.Content.(map[string]interface{})
				// log.Println("shm update received ", evt["Mem"].(map[string]interface{}))
				var ts time.Time
				json.Unmarshal([]byte(evt["Ts"].(string)), &ts)
				// if ts.After(this.Shm.LastUpdatedAt) {
				this.Shm.Update(evt["Mem"].(map[string]interface{}))
				// }
				// this.Shm.WriteVar(evt["Varname"].(string), evt["Value"], false) // not master
				// log.Println("updated", evt["Varname"], " to ", evt["Value"], " from ", msg.NodeID)
				// this.Shm.Update(newMem)

			default:
			}

		case <-this.interrupt:
			return
		}
	}

}

func (this *ClusterService) handleBroadcasts() {
	for {
		select {
		case msg := <-this.broadcastChan:
			for nodeID := range this.meshNet.outgoingChan {
				if msg.MsgType == bullyMsg {
					log.Println("bullying")
				}
				this.meshNet.outgoingChan[nodeID] <- msg
			}

		case shmUpdate := <-this.Shm.MasterChan:
			msg := Message{
				MsgType: shmMsg,
				NodeID:  this.meshNet.me.NodeID,
				Content: shmUpdate,
			}
			for nodeID := range this.meshNet.outgoingChan {
				this.meshNet.outgoingChan[nodeID] <- msg
			}

		case <-this.interrupt:
			return
		}
	}
	log.Println("manageIncomingMessages ends here")
}

func (this *ClusterService) handleSoldierDown(nodeID string) {
	log.Println("solider down ", nodeID)

	// if I'm the leader, I don't care if someone is down
	if !this.IsLeader {
		log.Println("leader election restarts")
		this.biggestBullySoFar = -1
		this.leaderElected = false
		this.lastBulliedAt = time.Now()
		this.bullyOthers()
	} else {
		log.Println("I'm the leader. Don't want a competitor.")
	}
}

func (this *ClusterService) handleBullyMsg(msg Message) {
	var val int = int(msg.Content.(float64))

	if val == this.meshNet.me.Priority {
		newPrio := rand.Intn(100)
		this.meshNet.me.Priority = newPrio
		log.Println(msg.NodeID, " has same priority. ", val, " Updating myself to ", newPrio)
		return
	}

	if val < this.meshNet.me.Priority {
		log.Println(this.meshNet.me.Priority, "bullying others")
		this.bullyOthers()
	} else {
		this.biggestBullySoFar = val
		this.lastBulliedAt = time.Now()
		this.IsLeader = false
		this.leaderElected = false
		log.Println(this.meshNet.me.Priority, " bullied by ", msg.NodeID, " with val ", val, " : ", this.biggestBullySoFar)
	}
}

func (this *ClusterService) bullyOthers() {
	this.IsLeader = false
	this.leaderElected = false
	this.lastBulliedAt = time.Now()
	this.broadcastChan <- Message{
		NodeID:  this.meshNet.me.NodeID,
		MsgType: bullyMsg,
		Content: this.meshNet.me.Priority,
	}
}

func (this *ClusterService) Start() {

	wg := sync.WaitGroup{}
	wg.Add(3)
	go this.meshNet.setupIncomingServer(this.clusterUrl, &wg)
	otherNodes := this.askDiscoveryServiceForPeers()
	log.Println("discovery: ", len(otherNodes), " peers found")
	go this.meshNet.setupOutgoingConn(otherNodes, &wg)
	go this.manageIncomingMessages(&wg)
	wg.Wait()
}

func (this *ClusterService) Shutdown() {
	// incoming server
	// outoingconn
	this.meshNet.Shutdown()
	// manageIncomingMessages
	this.interrupt <- true
	// handle broadcasts
	this.interrupt <- true
}

func (this *ClusterService) askDiscoveryServiceForPeers() []NodeInfoT {
	myinfo := this.meshNet.me
	b, _ := json.Marshal(myinfo)

	req, err := http.NewRequest("POST", this.discoveryUrl, bytes.NewBuffer(b))
	if err != nil {
		log.Fatal("failed to create newRequest err:", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		log.Panicln("failed to connect to discovery URL")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Fatal("discovery service didn't reply ok - ", res.StatusCode)
	}

	body, _ := ioutil.ReadAll(res.Body)
	var respObj map[string]string
	if err := json.Unmarshal(body, &respObj); err != nil {
		log.Fatal("cannot understand response from discovery service err:", err)
	}

	var otherNodes []NodeInfoT
	for nodeid, loc := range respObj {
		if nodeid != myinfo.NodeID {
			otherNodes = append(otherNodes, NodeInfoT{
				NodeID: nodeid,
				URL:    loc,
			})
		}
	}

	return otherNodes
}
