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
	Hall_reg [][]bool
	Cab_reg  []map[string]bool
}

func New_backup() *Backup {
	return &Backup{
		Hall_reg: utilities.Create_request_arr(config.N_floors, config.N_buttons),
		Cab_reg:  Create_cab_request_arr(config.N_floors),
	}
}

type Master struct {
	Client_list 	     map[string]*Elevator_client // key is addr
	Hall_requests        [][]bool
	Hall_assignments     [][]string
	Cab_requests         []map[string]bool
	Successor_addr       string
	Pending              map[string]*network.Message
	Resend_ticker        *time.Ticker
}

type Elevator_client struct {
	Connection 		*network.Client
	ID 				string
	Current_floor 	int
	Obstruction 	bool
	Busy 			bool
	Task_timer 		*time.Timer
}

func Create_cab_request_arr(floor int) []map[string]bool {
	cab_requests := make([]map[string]bool, floor)
	for f := 0; f < floor; f++ {
		cab_requests[f] = make(map[string]bool)
	}
	return cab_requests
}

func create_hall_assignment_arr(floors, buttons int) [][]string {
	assignments := make([][]string, floors)
	for f := 0; f < floors; f++ {
		assignments[f] = make([]string, buttons)
	}
	return assignments
}

func New_master() *Master {
	return &Master{
		Client_list :     make(map[string]*Elevator_client),
		Hall_requests:    utilities.Create_request_arr(config.N_floors, config.N_buttons),
		Hall_assignments: create_hall_assignment_arr(config.N_floors, config.N_buttons),
		Cab_requests:     Create_cab_request_arr(config.N_floors),
		Pending:          make(map[string]*network.Message),
		Resend_ticker:    time.NewTicker(config.Resend_rate),
	}
}

func (m *Master) Handle_message(message network.Message) {
	log.Printf("Recived from %s: %v", m.Client_list[message.Address].ID, message.Header)
	switch message.Header {
	case network.OrderReceived:
		if message.Payload.OrderButton == elevio.BT_Cab {
			m.Cab_requests[message.Payload.OrderFloor][m.Client_list[message.Address].ID] = true
		} else {
			m.Hall_requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
		}
		m.Client_list[m.Successor_addr].Send(network.Message{Header: network.Backup, Payload: &network.MessagePayload{BackupHall: m.Hall_requests, BackupCab: m.Cab_requests}, UID: message.UID})
		m.Pending[message.UID] = &message

	case network.OrderFulfilled:
		if message.Payload.OrderButton == elevio.BT_Cab {
			m.Cab_requests[message.Payload.OrderFloor][m.Client_list[message.Address].ID] = false
		} else {
			m.Hall_requests[message.Payload.OrderFloor][message.Payload.OrderButton] = false
			m.Hall_assignments[message.Payload.OrderFloor][message.Payload.OrderButton] = ""
		}
		
		if m.Still_busy(message.Address) {
			m.Client_list[message.Address].Busy = false
			m.Client_list[message.Address].Task_timer.Stop()
		} else {
			m.Client_list[message.Address].Task_timer.Reset(config.Request_timeout)
		}

		message.UID = utilities.Gen_uid(m.Client_list[m.Successor_addr].ID)
		m.Client_list[m.Successor_addr].Send(network.Message{Header: network.Backup, Payload: &network.MessagePayload{BackupHall: m.Hall_requests, BackupCab: m.Cab_requests}, UID: message.UID})
		m.Pending[message.UID] = &message

	case network.Ack:
		_, ok := m.Pending[message.UID]
		if ok {
			pend_msg := m.Pending[message.UID]
			switch pend_msg.Header {
			case network.OrderReceived:
				m.Distribute_request(pend_msg.Payload.OrderFloor, pend_msg.Payload.OrderButton, pend_msg.Address)
				m.Client_list[pend_msg.Address].Send(network.Message{Header: network.Ack, UID: pend_msg.UID})
				delete(m.Pending, pend_msg.UID)

			case network.OrderFulfilled:
				m.Send_light_update()
				delete(m.Pending, pend_msg.UID)
			}
		}

	case network.FloorUpdate:
		m.Client_list[message.Address].Current_floor = message.Payload.CurrentFloor

	case network.ObstructionUpdate:
		m.Client_list[message.Address].Obstruction = message.Payload.Obstruction
		if message.Payload.Obstruction {
			m.Client_list[message.Address].Task_timer.Stop()
		} else {
			if m.Client_list[message.Address].Busy {
				m.Client_list[message.Address].Task_timer.Reset(config.Request_timeout)
			}
		}

	case network.ClientInfo:
		m.Client_list[message.Address].ID = message.Payload.ID
		m.Client_list[message.Address].Current_floor = message.Payload.CurrentFloor
		m.Client_list[message.Address].Obstruction = message.Payload.Obstruction
		m.Resend_cab_request(message.Address)
	}
}