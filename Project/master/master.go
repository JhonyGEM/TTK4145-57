package master

import (
	"project/config"
	"project/elevio"
	"project/network"
	"project/utilities"
	"time"
	"log"
)

type Backup struct {
	HallRequests [][]bool
	CabRequests  []map[string]bool
}

func NewBackup() *Backup {
	return &Backup{
		HallRequests: utilities.NewRequests(),
		CabRequests:  NewCabRequests(),
	}
}

type Master struct {
	IP				    	string
	ClientList 	        	map[string]*ElevatorClient		// [address]
	Requests                MasterRequests
	HasSuccessor        	bool
	Successor               Successor
	Sequence            	int
	Pending             	map[string]*network.Pending		// [uid]
	Tickers                 MasterTickers
}

type MasterRequests struct {
	Hall					[][]bool						// [floor][button]
	HallAssignments     	[][]string						// [floor][button]
	Cab     		    	[]map[string]bool				// [floor][client id]
}

type Successor struct {
	Address					string
	IsNotified				bool
	SearchTimer   			*time.Timer
	SearchTimedOut     		bool	
}

type MasterTickers struct {
	Resend       			*time.Ticker
	Timeout					*time.Ticker
}

func NewCabRequests() []map[string]bool {
	cabRequests := make([]map[string]bool, config.N_floors)
	for floor := 0; floor < config.N_floors; floor++ {
		cabRequests[floor] = make(map[string]bool)
	}
	return cabRequests
}

func NewHallAssignments() [][]string {
	assignments := make([][]string, config.N_floors)
	for floor := 0; floor < config.N_floors; floor++ {
		assignments[floor] = make([]string, config.N_buttons)
	}
	return assignments
}

func NewMaster() *Master {
	master := &Master{
		ClientList:	make(map[string]*ElevatorClient),
		Requests: MasterRequests{
			Hall: 					utilities.NewRequests(), 
			HallAssignments: 		NewHallAssignments(), 
			Cab: 					NewCabRequests()},
		Successor: Successor{
			SearchTimer:			time.NewTimer(config.Successor_timeout),
		},
		Pending: make(map[string]*network.Pending),
		Tickers: MasterTickers{
			Resend: 				time.NewTicker(config.Resend_rate),
			Timeout: 				time.NewTicker(config.Timeout_check_rate),
		},
	}
	return master
}

func (m *Master) NotifyNewSuccessor() {
	for addr, client := range m.ClientList {
		if m.IP != client.Connection.GetIP() || (m.Successor.SearchTimedOut && m.IP == client.Connection.GetIP()) {
			message := network.Message{
				Header: network.Successor, 
				Address: addr,
				UID: utilities.GenUID("master", m.Sequence)}
			m.Sequence++
			m.Successor.IsNotified = true
			m.Pending[message.UID] = &network.Pending{Message: message, Timestamp: time.Now()}
			client.Send(message)
			return
		}
	}
	log.Println("No suitable successor found")
	m.Successor.SearchTimer.Reset(config.Successor_timeout)
}

func (m *Master) HandleMessage(message network.Message) {
	switch message.Header {
	case network.OrderReceived:
		if m.HasSuccessor {
			if !m.IsRequestActive(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address) {
				if message.Payload.OrderButton == elevio.BT_Cab {
					m.Requests.Cab[message.Payload.OrderFloor][m.ClientList[message.Address].ID] = true
				} else {
					m.Requests.Hall[message.Payload.OrderFloor][message.Payload.OrderButton] = true
				}
				m.Pending[message.UID] = &network.Pending{Message: message, Timestamp: time.Now()}
				m.ClientList[m.Successor.Address].Send(network.Message{
					Header: network.Backup, 
					Payload: &network.MessagePayload{BackupHall: m.Requests.Hall, BackupCab: m.Requests.Cab}, 
					UID: message.UID})
			} else {
				// Repeated request -> notify elevator
				m.ClientList[message.Address].Send(network.Message{
					Header: network.Ack, 
					UID: message.UID})
			}
		}

	case network.OrderFulfilled:
		if message.Payload.OrderButton == elevio.BT_Cab {
			m.Requests.Cab[message.Payload.OrderFloor][m.ClientList[message.Address].ID] = false
		} else {
			m.Requests.Hall[message.Payload.OrderFloor][message.Payload.OrderButton] = false
			m.Requests.HallAssignments[message.Payload.OrderFloor][message.Payload.OrderButton] = ""
		}
		if m.ClientList[message.Address].ActiveRequestCount > 0 {
			m.ClientList[message.Address].ActiveRequestCount--
		}
		
		if m.ClientList[message.Address].ActiveRequestCount > 0 {
			if !m.ClientList[message.Address].Obstruction {
				m.ClientList[message.Address].TaskTimer.Reset(config.Request_timeout)
			}
		} else {
			m.ClientList[message.Address].TaskTimer.Stop()
		}

		if m.HasSuccessor {
			message.UID = utilities.GenUID("master", m.Sequence)
			m.Sequence++
			m.Pending[message.UID] = &network.Pending{Message: message, Timestamp: time.Now()}
			m.ClientList[m.Successor.Address].Send(network.Message{
				Header: network.Backup, 
				Payload: &network.MessagePayload{BackupHall: m.Requests.Hall, BackupCab: m.Requests.Cab}, 
				UID: message.UID})
		} else {
			m.SendButtonLightUpdate()
		}

	case network.Ack:
		_, ok := m.Pending[message.UID]
		if ok {
			pend := m.Pending[message.UID].Message
			switch pend.Header {
			case network.OrderReceived:
				m.DistributeRequest(pend.Payload.OrderFloor, pend.Payload.OrderButton, pend.Address)
				m.ClientList[pend.Address].Send(network.Message{
					Header: network.Ack, 
					UID: pend.UID})

			case network.OrderFulfilled:
				m.SendButtonLightUpdate()
			
			case network.Successor:
				if !m.HasSuccessor {
					log.Printf("Successor found: %s", message.Address)
					m.HasSuccessor = true
					m.Successor.IsNotified = false
					m.Successor.SearchTimer.Stop()
					m.Successor.SearchTimedOut = false
					m.Successor.Address = message.Address
					m.ClientList[m.Successor.Address].Send(network.Message{
						Header: network.Backup, 
						Payload: &network.MessagePayload{BackupHall: m.Requests.Hall, BackupCab: m.Requests.Cab}, 
						UID: message.UID})
				}
			}
			delete(m.Pending, pend.UID)
		}

	case network.FloorUpdate:
		m.ClientList[message.Address].CurrentFloor = message.Payload.CurrentFloor

	case network.ObstructionUpdate:
		m.ClientList[message.Address].Obstruction = message.Payload.Obstruction
		if message.Payload.Obstruction {
			m.ClientList[message.Address].TaskTimer.Stop()
		} else {
			if m.ClientList[message.Address].ActiveRequestCount > 0 {
				m.ClientList[message.Address].TaskTimer.Reset(config.Request_timeout)
			}
		}

	case network.ClientInfo:
		m.ClientList[message.Address].ID = message.Payload.ID
		m.ClientList[message.Address].CurrentFloor = message.Payload.CurrentFloor
		m.ClientList[message.Address].Obstruction = message.Payload.Obstruction
		m.ResendCabRequest(message.Address)
	}
}

func (m *Master) HandleNewClient(client *network.Client) {
	m.AddClient(client)
	m.ClientList[client.Addr].Send(network.Message{Header: network.NotSuccessor})
	if !m.HasSuccessor && !m.Successor.IsNotified {
		m.NotifyNewSuccessor()
	}
	m.ResendCabRequest(client.Addr)
	m.SendButtonLightUpdate()
}

func (m *Master) HandleClientLoss(client *network.Client) {
	id := m.ClientList[client.Addr].ID
	m.RemoveClient(client.Addr)
	m.ResetHallAssignments(id)
	if len(m.ClientList) == 0 && !network.HasInternetConnection() {
		log.Fatal("Loss of internet")
	} 
	// Case when elevator client running on the same machine as master
	if len(m.ClientList) == 1 && !network.HasInternetConnection() {
		for _, client := range m.ClientList {
			client.Connection.Conn.Close()
			break
		}
	}
	if client.Addr == m.Successor.Address {
		m.HasSuccessor = false
		m.Successor.Address = ""
		if len(m.ClientList) > 0 {
			m.Successor.SearchTimer.Reset(config.Successor_timeout)
			m.NotifyNewSuccessor()
		}
	}
	if len(m.ClientList) > 0 {
		m.ResendHallRequest()
	}
}