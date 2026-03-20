package master

import (
	"project/config"
	"project/network"
	"log"
	"time"
)

type ElevatorClient struct {
	Connection 			*network.Client
	ID 					string
	CurrentFloor 		int
	Obstruction 		bool
	ActiveRequestCount  int
	TaskTimer 			*time.Timer
}

func (ec *ElevatorClient) Send(message network.Message) {
	if ec.Connection.Conn != nil {
		ec.Connection.Send(message)
	}
}

func (m *Master) AddClient(c *network.Client) {
	client := &ElevatorClient{
		Connection:     c,
		TaskTimer:      time.NewTimer(config.Request_timeout),
	}
	client.TaskTimer.Stop()
	m.ClientList[c.Addr] = client
	log.Printf("Client connected: %s", c.Addr)
}

func (m *Master) RemoveClient(addr string) {
	_, ok := m.ClientList[addr]
	if ok {
		delete(m.ClientList, addr)
		log.Printf("Client removed: %s", addr)
	}
}

func (m *Master) MonitorClientTasksTimers() {
	for addr, client := range m.ClientList {
		if client.TaskTimer == nil {
			continue
		}
		select {
		case <-client.TaskTimer.C:
			client.TaskTimer.Stop()
			log.Printf("Client timeout: %s, Task timer expired", client.ID)
			if m.Successor.Address == addr {
				client.Send(network.Message{Header: network.NotSuccessor})
			}
			client.Connection.Conn.Close()
		default:
		}
	}
}

func (m *Master) MonitorSuccessorAck() {
	for uid, pend := range m.Pending {
		if pend.Message.Header == network.Successor && time.Since(pend.Timestamp) > config.Pending_timeout {
			if !m.HasSuccessor {
				client, ok := m.ClientList[pend.Message.Address]
				if ok {
					client.Send(network.Message{Header: network.NotSuccessor})
				}
				m.Successor.IsNotified = false
				m.Successor.SearchTimer.Reset(config.Search_rate)
				m.NotifyNewSuccessor()
			}
			delete(m.Pending, uid)
		}
	}
}