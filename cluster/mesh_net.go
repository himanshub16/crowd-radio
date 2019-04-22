package cluster

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var upgrader websocket.Upgrader

type NodeInfoT struct {
	URL      string `json:"url"`
	NodeID   string `json:"node_id"`
	Priority int    `json:"priority"`
}

type MessageType string

const (
	bullyMsg     MessageType = "bullyMsg"
	shmMsg       MessageType = "shmMsg"
	heartbeatMsg MessageType = "heartbeatMsg"
)

type Message struct {
	NodeID  string      `json:"node_id"`
	MsgType MessageType `json:"message_type"`
	Content string      `json:"content"`
}

// There are some incoming connections and some outgoing connections
// All connections are persistent and websocket based
// The mesh manages channels for all individual connections
type MeshNetwork struct {
	// token obtained from discovery service to secure cluster
	authToken string
	me        NodeInfoT

	// server to handle incoming connections
	incomingServer http.Server
	incomingMux    *http.ServeMux

	// channels to send and recieve events
	chanMutex          *sync.Mutex
	outgoingChan       map[string](chan Message)
	commonIncomingChan chan Message

	broadcastChan chan Message

	// interrupt channel for each connection
	interruptConnChan    map[string](chan interface{})
	interruptServiceChan chan interface{}
}

func NewMeshNetwork(me NodeInfoT, authToken string) *MeshNetwork {
	return &MeshNetwork{
		authToken: authToken,
		me:        me,

		incomingServer: http.Server{
			Addr: me.URL,
		},
		incomingMux: nil,

		chanMutex:          &sync.Mutex{},
		outgoingChan:       make(map[string](chan Message)),
		commonIncomingChan: make(chan Message),

		interruptConnChan:    make(map[string](chan interface{})),
		interruptServiceChan: make(chan interface{}),
	}
}

func (this *MeshNetwork) setupOutgoingConn(nodes []NodeInfoT, parentWg *sync.WaitGroup) {
	defer parentWg.Done()

	wg := sync.WaitGroup{}
	for _, node := range nodes {
		wg.Add(1)
		go this.setupOutgoingToSingleNode(node, &wg)
	}
	wg.Wait()
}

func (this *MeshNetwork) setupOutgoingToSingleNode(node NodeInfoT, wg *sync.WaitGroup) {
	defer wg.Done()

	var addr, nodeID string
	addr = fmt.Sprint("ws://", node.URL)
	nodeID = node.NodeID

	hdr := make(http.Header)
	hdr.Add("auth_token", this.authToken)
	hdr.Add("node_id", this.me.NodeID)

	log.Println("connecting to ", addr)
	conn, _, err := websocket.DefaultDialer.Dial(addr, hdr)
	if err != nil {
		log.Println("dial ws: ", addr, " err: ", err)
		return
	}

	// setup required channels, and ensure their cleanup
	this.openChannelsForNewNode(nodeID)
	defer this.closeChannelsForNode(nodeID)

	defer conn.Close()
	// receive messages
	go func() {
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				log.Println("failed to read msg:", err)
				this.interruptConnChan[nodeID] <- true
				return
			}
			this.commonIncomingChan <- msg
		}
	}()

	// send messages
	for {
		select {
		// some message to send
		case msg := <-this.outgoingChan[nodeID]:
			if err := conn.WriteJSON(msg); err != nil {
				log.Println("failed to send message to ", nodeID, " err:", err)
				return
			}

		// signalled to interrupt
		case <-this.interruptConnChan[nodeID]:
			if err := conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {

				log.Println("failed to close ws conn for ", nodeID, " err:", err)
				return
			}
		}
	}
}

func (this *MeshNetwork) setupIncomingServer(addr string, parentWg *sync.WaitGroup) {
	defer parentWg.Done()

	this.incomingMux = http.NewServeMux()
	this.incomingMux.HandleFunc("/health", this.healthCheckHandler)
	this.incomingMux.HandleFunc("/", this.handleIncomingConn)

	this.incomingServer.Handler = this.incomingMux
	log.Println("listening at ", this.incomingServer.Addr)
	if err := this.incomingServer.ListenAndServe(); err != nil {
		log.Fatal("cannot start incoming server err:", err)
	}
}

func (this *MeshNetwork) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("auth_token")
	if token != this.authToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing/incorrect auth_token header")
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Good. Thanks for asking!")
}

func (this *MeshNetwork) handleIncomingConn(w http.ResponseWriter, r *http.Request) {
	// TODO checks for token
	var token, nodeID string
	token = r.Header.Get("auth_token")
	nodeID = r.Header.Get("node_id")

	if token != this.authToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "Missing/incorrect auth_token header")
		return
	}

	if _, exists := this.interruptConnChan[nodeID]; exists {
		w.WriteHeader(http.StatusAlreadyReported)
		fmt.Fprint(w, "Already connected")
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("failed to upgrade ws", err)
	}
	defer ws.Close()

	// setup reuqired channels, ensuring their cleanup
	this.openChannelsForNewNode(nodeID)
	defer this.closeChannelsForNode(nodeID)
	log.Println("new peer:", nodeID)

	// receive messages
	go func() {
		for {
			var msg Message
			if err := ws.ReadJSON(&msg); err != nil {
				log.Println("failed reading message ", err)
				this.interruptConnChan[nodeID] <- true
				break
			}
			this.commonIncomingChan <- msg
		}
	}()

	// send messages
	for {
		select {
		case msg := <-this.outgoingChan[nodeID]:
			if err := ws.WriteJSON(msg); err != nil {
				log.Println("sender failed to send message to", nodeID, " err:", err)
				return
			}

		case <-this.interruptConnChan[nodeID]:
			return
		}
	}
}

func (this *MeshNetwork) openChannelsForNewNode(nodeID string) {
	this.chanMutex.Lock()
	this.outgoingChan[nodeID] = make(chan Message)
	this.interruptConnChan[nodeID] = make(chan interface{})
	this.chanMutex.Unlock()
}

func (this *MeshNetwork) closeChannelsForNode(nodeID string) {
	this.chanMutex.Lock()
	close(this.outgoingChan[nodeID])
	delete(this.outgoingChan, nodeID)
	close(this.interruptConnChan[nodeID])
	delete(this.interruptConnChan, nodeID)
	this.chanMutex.Unlock()
	fmt.Println("connection closed for ", nodeID)
}