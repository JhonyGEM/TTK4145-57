package master

import (
	"project/config"
	"project/elevio"
	"project/network"
	"project/utilities"
	"time"
)

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
	Connection    *network.Client
	ID            string
	Current_floor int
	Obstruction   bool
	Busy          bool
	Task_timer    *time.Timer
}

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
		Client_list:      make(map[string]*Elevator_client),
		Hall_requests:    utilities.Create_request_arr(config.N_floors, config.N_buttons),
		Hall_assignments: create_hall_assignments(config.N_floors, config.N_buttons),
		Cab_requests:     Create_cab_requests(config.N_floors),
		Pending:          make(map[string]*network.Message),
		Resend_ticker:    time.NewTicker(config.Resend_rate),
	}
}

func (m *Master) Send_light_update() {
	for _, elev := range m.Client_list {
		light := utilities.Create_request_arr(config.N_floors, config.N_buttons)
		for f := 0; f < config.N_floors; f++ {
			copy(light[f], m.Hall_requests[f])
			light[f][elevio.BT_Cab] = m.Cab_requests[f][elev.ID]
		}
		elev.Send(network.Message{Header: network.LightUpdate, Payload: &network.DataPayload{Lights: light}})
	}
}
