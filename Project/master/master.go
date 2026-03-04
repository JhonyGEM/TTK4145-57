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
		HallRequests: utilities.NewRequests(config.N_floors, config.N_buttons),
		CabRequests:  NewCabRequests(config.N_floors),
	}
}

type Master struct {
	IP				    string
	ClientList 	        map[string]*ElevatorClient	// key = address
	HallRequests        [][]bool
	HallAssignments     [][]string
	CabRequests         []map[string]bool
	HasSuccessor        bool
	SuccessorAddr       string
	Sequence            int
	Pending             map[string]*network.Message	// key = uid
	ResendTicker        *time.Ticker
}

type ElevatorClient struct {
	Connection 		*network.Client
	ID 				string
	CurrentFloor 	int
	PreviousFloor   int
	Obstruction 	bool
	ActiveReq       int
	TaskTimer 		*time.Timer
}

func NewCabRequests(floor int) []map[string]bool {
	cab_requests := make([]map[string]bool, floor)
	for f := 0; f < floor; f++ {
		cab_requests[f] = make(map[string]bool)
	}
	return cab_requests
}

func NewHallAssignments(floors, buttons int) [][]string {
	assignments := make([][]string, floors)
	for f := 0; f < floors; f++ {
		assignments[f] = make([]string, buttons)
	}
	return assignments
}

func NewMaster() *Master {
	return &Master{
		ClientList :      make(map[string]*ElevatorClient),
		HallRequests:     utilities.NewRequests(config.N_floors, config.N_buttons),
		HallAssignments:  NewHallAssignments(config.N_floors, config.N_buttons),
		CabRequests:      NewCabRequests(config.N_floors),
		Pending:          make(map[string]*network.Message),
		ResendTicker:     time.NewTicker(config.Resend_rate),
	}
}

func (m *Master) FindNewSuccessor() {
	for _, client := range m.ClientList {
		if m.IP != client.Connection.GetIP() {
			m.HasSuccessor = true
			m.SuccessorAddr = client.Connection.Addr
			client.Send(network.Message{Header: network.Succesor})
			break
		}
	}
	log.Print("No suitable successor found \n")
}

func (m *Master) HandleMessage(message network.Message) {
	//log.Printf("Recived from %s: %v", m.Client_list[message.Address].ID, message.Header)
	switch message.Header {
	case network.OrderReceived:
		if m.HasSuccessor {
			if !m.IsRequestActive(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address) {
				if message.Payload.OrderButton == elevio.BT_Cab {
					m.CabRequests[message.Payload.OrderFloor][m.ClientList[message.Address].ID] = true
				} else {
					m.HallRequests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
				}
				m.Pending[message.UID] = &message
				m.ClientList[m.SuccessorAddr].Send(network.Message{Header: network.Backup, 
																	 Payload: &network.MessagePayload{BackupHall: m.HallRequests, BackupCab: m.CabRequests}, 
																	 UID: message.UID})
			} else {
				// Repeated request -> send ack to elevator to remove from pending list
				m.ClientList[message.Address].Send(network.Message{Header: network.Ack, 
																	UID: message.UID})
			}
		}

	case network.OrderFulfilled:
		if message.Payload.OrderButton == elevio.BT_Cab {
			m.CabRequests[message.Payload.OrderFloor][m.ClientList[message.Address].ID] = false
		} else {
			m.HallRequests[message.Payload.OrderFloor][message.Payload.OrderButton] = false
			m.HallAssignments[message.Payload.OrderFloor][message.Payload.OrderButton] = ""
		}
		m.ClientList[message.Address].ActiveReq--
		
		if m.ClientList[message.Address].ActiveReq > 0 {
			if !m.ClientList[message.Address].Obstruction {
				m.ClientList[message.Address].TaskTimer.Reset(config.Request_timeout)
			}
		} else {
			m.ClientList[message.Address].TaskTimer.Stop()
		}

		if m.HasSuccessor {
			message.UID = utilities.GenUID("master", m.Sequence)
			m.Sequence++
			m.ClientList[m.SuccessorAddr].Send(network.Message{Header: network.Backup, 
																Payload: &network.MessagePayload{BackupHall: m.HallRequests, BackupCab: m.CabRequests}, 
																UID: message.UID})
			m.Pending[message.UID] = &message
		}

	case network.Ack:
		_, ok := m.Pending[message.UID]
		if ok {
			pend := m.Pending[message.UID]
			switch pend.Header {
			case network.OrderReceived:
				m.DistributeRequest(pend.Payload.OrderFloor, pend.Payload.OrderButton, pend.Address)
				m.ClientList[pend.Address].Send(network.Message{Header: network.Ack, 
																	 UID: pend.UID})
				delete(m.Pending, pend.UID)

			case network.OrderFulfilled:
				m.SendLightUpdate()
				delete(m.Pending, pend.UID)
			}
		}

	case network.FloorUpdate:
		m.ClientList[message.Address].PreviousFloor = m.ClientList[message.Address].CurrentFloor
		m.ClientList[message.Address].CurrentFloor = message.Payload.CurrentFloor

	case network.ObstructionUpdate:
		m.ClientList[message.Address].Obstruction = message.Payload.Obstruction
		if message.Payload.Obstruction {
			m.ClientList[message.Address].TaskTimer.Stop()
		} else {
			if m.ClientList[message.Address].ActiveReq > 0 {
				m.ClientList[message.Address].TaskTimer.Reset(config.Request_timeout)
			}
		}

	case network.ClientInfo:
		m.ClientList[message.Address].ID = message.Payload.ID
		m.ClientList[message.Address].CurrentFloor = message.Payload.CurrentFloor
		m.ClientList[message.Address].PreviousFloor = message.Payload.CurrentFloor
		m.ClientList[message.Address].Obstruction = message.Payload.Obstruction
		m.ResendCabRequest(message.Address)
	}
	//m.Print()
}