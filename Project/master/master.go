package master

import (
	"config"
	"elevio"
	"log"
	"network"
	"time"
	"utilities"
)

type Master struct {
	Client_list 	     map[string]*Elevator_client // key is addr
	Hall_requests        [][]bool
	Hall_assignments     [][]string
	Cab_requests         []map[string]bool
	Successor_addr       string
	Pending              map[string]*network.Message
}

type Elevator_client struct {
	Connection 		*network.Client
	ID 				string
	Current_floor 	int
	Obstruction 	bool
	Busy 			bool
	Task_timer 		*time.Timer
}

// TODO: Need to implement
// 1. Better file division and variable/function names
// 2. Redundancy in communication / sync of hall and cab request from master (maybe WAL or loging of messages?)
// 4. Resend order after new master takeover after x seconds
// 5. Test if everyting thats implemented works

// Communication redundancy
// each event (new reques) need an unique id
// when elevator recieves request it will send it to master and save it pending array until it gets ack, it will be resedn if no ack in certain time window
// master will recive the message, handle the event, send copy to sucsessor and wait for ack before sending ack to elevator
//      maybe add local saving of the pending array
//      the pending array will be of type message and will be keyed by event id (time at the event was send + id of client)



func Create_cab_requests(floor int) []map[string]bool {
	cab_requests := make([]map[string]bool, floor)
	for f := 0; f < floor; f++ {
		cab_requests[f] = make(map[string]bool)
	}
	return cab_requests
}

func create_hall_assignments(floors, buttons int) [][]string {
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
		Hall_assignments: create_hall_assignments(config.N_floors, config.N_buttons),
		Cab_requests:     Create_cab_requests(config.N_floors),
		Pending:          make(map[string]*network.Message),
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

func (m *Master) send_light_update() {
	for _, elev := range m.Client_list {
		elev.Connection.Send(network.Message{Header: network.LightUpdate, Payload: &network.DataPayload{Lights: m.Hall_requests}})
	}
}

func (m *Master) find_closest_elevator(floor int) string {
	closest_addr := ""
	shortest_dist := 9999

	for _, client := range m.Client_list {
		distance := utilities.Abs(client.Current_floor - floor)
		if client.Busy {
			distance += config.Busy_penalty
		}
		if distance < shortest_dist {
			closest_addr = client.Connection.Addr
			shortest_dist = distance
		}
	}
	return closest_addr
}

func (m *Master) Distribute_request(floor int, button elevio.ButtonType, addr string) {
	if button == elevio.BT_Cab {
		m.Client_list[addr].Busy = true
		m.Client_list[addr].Connection.Send(network.Message{Header: network.OrderReceived, Payload: &network.DataPayload{OrderFloor: floor, OrderButton: button}})
		m.Client_list[addr].Task_timer.Reset(config.Request_timeout)
	} else {
		client_addr := m.find_closest_elevator(floor)
		if client_addr != "" {
			m.Hall_assignments[floor][button] = m.Client_list[client_addr].ID
			m.Client_list[client_addr].Busy = true
			m.Client_list[client_addr].Connection.Send(network.Message{Header: network.OrderReceived, Payload: &network.DataPayload{OrderFloor: floor, OrderButton: button}})
			m.send_light_update()
			m.Client_list[client_addr].Task_timer.Reset(config.Request_timeout)
		}
	}
}

func (m *Master) Redistribute_request(id string) {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if m.Hall_assignments[f][b] == id {
				m.Distribute_request(f, b, "")
			}
		}
	}
}

func (m *Master) Should_clear_busy(addr string) bool {
	for f := 0; f < config.N_floors; f++ {
		if m.Cab_requests[f][m.Client_list[addr].ID] {
			return false
		}	
	}
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if m.Hall_assignments[f][b] == m.Client_list[addr].ID {
				return false
			}
		}
	}
	return true
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
