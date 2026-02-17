package master

import (
	"project/config"
	"project/network"
	"log"
	"time"
)

func (ec *Elevator_client) Send(message network.Message) {
	if ec.Connection != nil {
		ec.Connection.Send(message)
	}
}

func (m *Master) Add_client(c *network.Client) {
	elev := &Elevator_client{
		Connection:    c,
		Current_floor: 0,
		Obstruction:   false,
		Busy:          false,
		Task_timer:    time.NewTimer(config.Request_timeout),
	}
	elev.Task_timer.Stop()
	m.Client_list[c.Addr] = elev
	log.Printf("Client connected: %s", c.Addr)
}

func (m *Master) Remove_client(addr string) {
	_, ok := m.Client_list[addr]
	if ok {
		delete(m.Client_list, addr)
		log.Printf("Client removed: %s", addr)
	}
}

func (m *Master) Client_timer_handler(timeout chan<- *network.Client) {
	for {
		for _, client := range m.Client_list {
			if client.Task_timer != nil {
				select {
				case <-client.Task_timer.C:
					client.Task_timer.Stop()
					timeout<- client.Connection

				default:
				
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}