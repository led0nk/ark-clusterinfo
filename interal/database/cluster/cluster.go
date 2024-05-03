package cluster

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/FlowingSPDG/go-steam"
)

type Cluster struct {
	filename string
	server   map[string]*Server
	mu       sync.Mutex
}

type Server struct {
	name        string
	addr        string
	serverInfo  *steam.InfoResponse
	playersInfo *steam.PlayersInfoResponse
}

func NewCluster(filename string) (*Cluster, error) {
	cluster := &Cluster{
		filename: filename,
		server:   make(map[string]*Server),
	}
	if err := cluster.readJSON(); err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *Cluster) writeJSON() error {

	as_json, err := json.MarshalIndent(c.server, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile(c.filename, as_json, 0644)
	if err != nil {
		return err
	}
	return nil
}

// read JSON data from file = filename
func (c *Cluster) readJSON() error {

	if _, err := os.Stat(c.filename); os.IsNotExist(err) {
		return errors.New("file does not exist")
	}
	data, err := os.ReadFile(c.filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &c.server)
}

func (c *Cluster) CreateServer(server *Server) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if server.name == "" {
		server.name = "newServer"
	}

	c.server[server.name] = server
	if err := c.writeJSON(); err != nil {
		return "", err
	}

	return server.name, nil
}

func (c *Cluster) GetUserByName(name string) (*Server, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if name == "" {
		return nil, errors.New("empty name")
	}

	fetchedServer := &Server{}
	for _, server := range c.server {
		if server.name == name {
			fetchedServer = server
		}
	}
	return fetchedServer, nil
}

func (c *Cluster) DeleteServer(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if name == "" {
		return errors.New("requires server name")
	}

	if _, exists := c.server[name]; !exists {
		return errors.New("server doesn't exist")
	}

	delete(c.server, name)

	if err := c.writeJSON(); err != nil {
		return err
	}
	return nil
}
