package client

import (
	"fmt"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type ClientManager struct {
	clients     map[string]*Client // 使用host:port作为key
	clientsLock sync.RWMutex
	register    chan *Client
}

func NewClientManager() *ClientManager {
	m := &ClientManager{
		clients:  make(map[string]*Client),
		register: make(chan *Client, 1000),
	}

	go m.startListener()
	go m.statLoop()

	return m
}

func (m *ClientManager) startListener() {
	for client := range m.register {
		m.RegisterClient(client)
	}
}

func (m *ClientManager) RegisterClient(client *Client) {
	key := m.getKey(client.GetHost(), client.GetPort())
	m.clientsLock.Lock()
	defer m.clientsLock.Unlock()

	if _, exists := m.clients[key]; exists {
		logx.Errorf("Client already registered: %s:%d", client.GetHost(), client.GetPort())
		return
	}

	m.clients[key] = client
	logx.Infof("Registered new client: %s:%d", client.GetHost(), client.GetPort())
}

func (m *ClientManager) UnregisterClient(client *Client) {
	key := m.getKey(client.GetHost(), client.GetPort())
	m.clientsLock.Lock()
	defer m.clientsLock.Unlock()

	delete(m.clients, key)
}

func (m *ClientManager) GetClient(host string, port int) (*Client, error) {
	key := m.getKey(host, port)
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()

	client, ok := m.clients[key]
	if !ok {
		return nil, fmt.Errorf("client not found: %s:%d", host, port)
	}

	return client, nil
}

func (m *ClientManager) GetClientOrNil(host string, port int) (*Client, error) {
	client, err := m.GetClient(host, port)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (m *ClientManager) GetClients() map[*Client]bool {
	clients := make(map[*Client]bool)
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()

	for _, client := range m.clients {
		clients[client] = true
	}

	return clients
}

func (m *ClientManager) GetAllClients() []*Client {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()

	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	return clients
}

func (m *ClientManager) GetClientCount() int {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()

	return len(m.clients)
}

func (m *ClientManager) PublishRegister(client *Client) {
	m.register <- client
}

func (m *ClientManager) getKey(host string, port int) string {
	return fmt.Sprintf("%s:%d", host, port)
}

func (m *ClientManager) statLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.printStats()
	}
}

func (m *ClientManager) printStats() {
	m.clientsLock.RLock()
	defer m.clientsLock.RUnlock()

	total := len(m.clients)
	connected := 0
	disconnected := 0

	for _, client := range m.clients {
		if client.IsConnected() {
			connected++
		} else {
			disconnected++
		}
	}

	logx.Statf("client_manager(iec104) - total_clients: %d, connected: %d, disconnected: %d",
		total, connected, disconnected)
}
