package master

import (
	"project/config"
	"project/network"
	"log"
	"time"
)

func (ec *ElevatorClient) Send(message network.Message) {
	if ec.Connection.Conn != nil {
		ec.Connection.Send(message)
	}
}

func (m *Master) AddClient(c *network.Client) {
	elev := &ElevatorClient{
		Connection:     c,
		TaskTimer:      time.NewTimer(config.Request_timeout),
	}
	elev.TaskTimer.Stop()
	m.ClientList[c.Addr] = elev
	log.Printf("Client connected: %s", c.Addr)
}

func (m *Master) RemoveClient(addr string) {
	_, ok := m.ClientList[addr]
	if ok {
		delete(m.ClientList, addr)
		log.Printf("Client removed: %s", addr)
	}
}

// Handles task timer for each client
func (m *Master) ClientTimerHandler() {
	for {
		for _, client := range m.ClientList {
			if client.TaskTimer != nil {
				select {
				case <-client.TaskTimer.C:
					client.TaskTimer.Stop()
					client.Connection.Conn.Close()

				default:
				
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}