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

func (m *Master) MonitorTimeouts() {
	ticker := time.NewTicker(config.MonitorInterval)
	for {
		select {
		case <-ticker.C:
			for _, client := range m.ClientList {
				if client.TaskTimer == nil {
					continue
				}
				select {
				case <-client.TaskTimer.C:
					client.TaskTimer.Stop()
					client.Connection.Conn.Close()
				default:
				}
			}
			for uid, pend := range m.Pending {
				if pend.Message.Header == network.Successor && time.Since(pend.Timestamp) > config.Pending_timeout {
					m.ProspectNotified = false
					delete(m.Pending, uid)
					m.FindNewSuccessor()
				}
			}
		}
	}
}