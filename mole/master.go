package mole

import (
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type Master struct {
	sync.RWMutex                          // protect agents map
	agents       map[string]*ClusterAgent // agents held all of joined agents
	listen       string                   // listen address
	authToken    string                   // TODO auth token
	heartbeat    time.Duration            // TODO heartbeat interval to ping agents
}

func NewMaster(cfg *Config) *Master {
	return &Master{
		listen:    cfg.Listen,
		authToken: "xxx",
		heartbeat: time.Second * 60,
		agents:    make(map[string]*ClusterAgent),
	}
}

func (m *Master) Serve() error {
	l, err := net.Listen("tcp", m.listen)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("master Accept error: %v", err)
			return err
		}

		go m.handle(conn)
	}

	return nil
}

func (m *Master) handle(conn net.Conn) {
	cmd, err := NewDecoder(conn).Decode()
	if err != nil {
		log.Printf("master decode protocol error: %v", err)
		return
	}

	if err := cmd.valid(); err != nil {
		log.Printf("master received invalid command: %v", err)
		return
	}

	switch cmd.Cmd {

	case cmdJoin:
		log.Println("agent joined", cmd.AgentID)
		m.AddAgent(cmd.AgentID, conn) // this is the persistent control connection

	case cmdNewWorker:
		log.Println("agent new worker connection", cmd.WorkerID)
		pub.Publish(&clusterWorker{
			agentID:       cmd.AgentID,
			workerID:      cmd.WorkerID,
			conn:          conn, // this is the worker connection
			establishedAt: time.Now(),
		})
		m.FreshAgent(cmd.AgentID)

	case cmdLeave: // FIXME better within controll conn instead of here
		log.Println("agent leaved", cmd.AgentID)
		m.CloseAgent(cmd.AgentID)

	case cmdPing: // FIXME better within control conn instead of here
		log.Println("agent heartbeat", cmd.AgentID)
		m.FreshAgent(cmd.AgentID)

	}
}

func (m *Master) AddAgent(id string, conn net.Conn) {
	m.Lock()
	defer m.Unlock()
	// if we already have agent connection with the same id
	// close the previous staled connection and use the new one
	if agent, ok := m.agents[id]; ok {
		agent.conn.Close()
	}

	ca := &ClusterAgent{
		id:         id,
		conn:       conn,
		joinAt:     time.Now(),
		lastActive: time.Now(),
	}

	m.agents[id] = ca
}

func (m *Master) CloseAgent(id string) {
	m.Lock()
	defer m.Unlock()
	if agent, ok := m.agents[id]; ok {
		agent.conn.Close()
		delete(m.agents, id)
	}
}

func (m *Master) FreshAgent(id string) {
	m.Lock()
	defer m.Unlock()
	if agent, ok := m.agents[id]; ok {
		agent.lastActive = time.Now()
	}
}

// the caller should check the returned ClusterAgent is not nil
// otherwise the agent hasn't connected to the cluster
func (m *Master) Agent(id string) *ClusterAgent {
	m.RLock()
	defer m.RUnlock()
	return m.agents[id]
}

func (m *Master) Agents() map[string]*ClusterAgent {
	m.RLock()
	defer m.RUnlock()
	return m.agents
}

//
// ClusterAgent is a runtime agent object within master lifttime
type ClusterAgent struct {
	id         string   // agent id
	conn       net.Conn // persistent control connection
	joinAt     time.Time
	lastActive time.Time
}

// Dial specifies the dial function for creating unencrypted TCP connections within the http.Client
func (ca *ClusterAgent) Dial(network, addr string) (net.Conn, error) {
	wid := randNumber(10)

	// notify the agent to create a new worker connection
	command := newCmd(cmdNewWorker, ca.id, wid)
	_, err := ca.conn.Write(command)
	if err != nil {
		return nil, err
	}

	// subcribe waitting for the worker id connection
	sub := pub.Subcribe(func(v interface{}) bool {
		if vv, ok := v.(*clusterWorker); ok {
			return vv.workerID == wid && vv.agentID == ca.id
		}
		return false
	})
	defer pub.Evict(sub) // evict the subcriber before exit

	select {
	case cw := <-sub:
		return cw.(*clusterWorker).conn, nil
	case <-time.After(time.Second * 10):
		return nil, errors.New("agent Dial(): new worker conn timeout")
	}

	return nil, errors.New("never be here")
}

// Client obtain a http client for an agent with customized dialer
func (ca *ClusterAgent) Client() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: ca.Dial,
		},
	}
}

//
// clusterWorker is a worker connection
type clusterWorker struct {
	agentID       string
	workerID      string
	conn          net.Conn
	establishedAt time.Time
}
