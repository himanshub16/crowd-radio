package cluster

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type ClusterService struct {
	clusterUrl   string
	discoveryUrl string
	meshNet      *MeshNetwork
}

func NewClusterService(clusterUrl, discoveryUrl string, me NodeInfoT, authToken string) *ClusterService {
	return &ClusterService{
		clusterUrl:   clusterUrl,
		discoveryUrl: discoveryUrl,
		meshNet:      NewMeshNetwork(me, authToken),
	}
}

func (this *ClusterService) manageIncomingMessages(parentwg *sync.WaitGroup) {
	go func() {
		defer parentwg.Done()
		ticker := time.NewTicker(time.Second * 1)
		for range ticker.C {
			for nodeID := range this.meshNet.outgoingChan {
				msg := Message{
					NodeID:  this.meshNet.me.NodeID,
					MsgType: bullyMsg,
					Content: "hello",
				}

				this.meshNet.outgoingChan[nodeID] <- msg
			}
		}

		log.Println("manageIncomingMessages ends here")
	}()

	for msg := range this.meshNet.commonIncomingChan {
		log.Println("incoming message", msg)
	}
}

func (this *ClusterService) Start() {

	wg := sync.WaitGroup{}
	wg.Add(3)
	go this.meshNet.setupIncomingServer(this.clusterUrl, &wg)
	otherNodes := this.askDiscoveryServiceForPeers()
	go this.meshNet.setupOutgoingConn(otherNodes, &wg)
	go this.manageIncomingMessages(&wg)
	wg.Wait()
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
