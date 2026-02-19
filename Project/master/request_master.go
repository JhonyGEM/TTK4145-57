package master

import (
	"project/config"
	"project/elevio"
	"project/utilities"
	"project/network"
)

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
		m.Client_list[addr].Send(network.Message{Header: network.OrderReceived, Payload: &network.DataPayload{OrderFloor: floor, OrderButton: button}})
		m.Client_list[addr].Task_timer.Reset(config.Request_timeout)
	} else {
		client_addr := m.find_closest_elevator(floor)
		if client_addr != "" {
			m.Hall_assignments[floor][button] = m.Client_list[client_addr].ID
			m.Client_list[client_addr].Busy = true
			m.Client_list[client_addr].Send(network.Message{Header: network.OrderReceived, Payload: &network.DataPayload{OrderFloor: floor, OrderButton: button}})
			m.Client_list[client_addr].Task_timer.Reset(config.Request_timeout)
		}
	}
	m.Send_light_update()
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

func (m *Master) Resend_cab_request(addr string) {
	id := m.Client_list[addr].ID
	for f := 0; f < config.N_floors; f++ {
		if m.Cab_requests[f][id] {
			m.Distribute_request(f, elevio.BT_Cab, addr)
		}
	}
}

func (m *Master) Resend_hall_request() {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(1); b++ {
			if m.Hall_requests[f][b] && m.Hall_assignments[f][b] == "" {
				m.Distribute_request(f, b, "")
			}
		}
	}
}

func (m *Master) Still_busy(addr string) bool {
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