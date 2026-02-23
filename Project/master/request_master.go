package master

import (
	"project/config"
	"project/elevio"
	"project/utilities"
	"project/network"
)

func (m *Master) select_optimal_elevator(floor int) string {
	chosen_addr := ""
	lowest_cost := 9999

	for _, client := range m.Client_list {
		curr_floor := client.Current_floor
		cost := utilities.Abs(curr_floor - floor) + client.Active_req * config.Cost_penalty
		
		if client.Active_req > 0 {
			for f := 0; f < config.N_floors; f++ {
				for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
					if m.Hall_requests[f][b] && m.Hall_assignments[f][b] == client.ID {
						cost += utilities.Abs(curr_floor - f)
						curr_floor = f
					}
				}
			}
		}

		if cost < lowest_cost {
			chosen_addr = client.Connection.Addr
			lowest_cost = cost
		}
	}
	//log.Printf("Cost: %d, chosen client: %s \n", lowest_cost, m.Client_list[chosen_addr].ID)
	return chosen_addr
}

func (m *Master) Is_request_active(floor int, button elevio.ButtonType, addr string) bool {
	if button == elevio.BT_Cab {
		return m.Cab_requests[floor][m.Client_list[addr].ID]
	} else {
		return m.Hall_requests[floor][button]
	}
}

func (m *Master) Distribute_request(floor int, button elevio.ButtonType, addr string) {
	if button == elevio.BT_Cab {
		m.Client_list[addr].Active_req++
		m.Client_list[addr].Send(network.Message{Header: network.OrderReceived, 
												 Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
		if !m.Client_list[addr].Obstruction {
			m.Client_list[addr].Task_timer.Reset(config.Request_timeout)
		}
	} else {
		client_addr := m.select_optimal_elevator(floor)
		if client_addr != "" {
			m.Hall_assignments[floor][button] = m.Client_list[client_addr].ID
			m.Client_list[client_addr].Active_req++
			m.Client_list[client_addr].Send(network.Message{Header: network.OrderReceived, 
															Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
			if !m.Client_list[client_addr].Obstruction {
				m.Client_list[client_addr].Task_timer.Reset(config.Request_timeout)
			}
		}
	}
	m.Send_light_update()
}

func (m *Master) Redistribute_hall_request(id string) {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
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
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
			if m.Hall_requests[f][b] && m.Hall_assignments[f][b] == "" {
				m.Distribute_request(f, b, "")
			}
		}
	}
}

func (m *Master) Send_light_update() {
	for _, elev := range m.Client_list {
		light := utilities.Create_request_arr(config.N_floors, config.N_buttons)
		for f := 0; f < config.N_floors; f++ {
			copy(light[f], m.Hall_requests[f])
			light[f][elevio.BT_Cab] = m.Cab_requests[f][elev.ID]
		}
		elev.Send(network.Message{Header: network.LightUpdate, 
								  Payload: &network.MessagePayload{Lights: light}})
	}
}