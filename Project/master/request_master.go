package master

import (
	"project/config"
	"project/elevio"
	"project/utilities"
	"project/network"
)

func (m *Master) selectOptimalElevator(floor int) string {
	chosen_addr := ""
	lowest_cost := 9999

	for _, client := range m.ClientList {
		curr_floor := client.CurrentFloor
		cost := utilities.Abs(curr_floor - floor) + client.ActiveReq * config.Cost_penalty
		
		if client.ActiveReq > 0 {
			for f := 0; f < config.N_floors; f++ {
				for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
					if m.HallRequests[f][b] && m.HallAssignments[f][b] == client.ID {
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

func (m *Master) IsRequestActive(floor int, button elevio.ButtonType, addr string) bool {
	if button == elevio.BT_Cab {
		return m.CabRequests[floor][m.ClientList[addr].ID]
	} else {
		return m.HallRequests[floor][button]
	}
}

func (m *Master) DistributeRequest(floor int, button elevio.ButtonType, addr string) {
	if button == elevio.BT_Cab {
		m.ClientList[addr].ActiveReq++
		m.ClientList[addr].Send(network.Message{Header: network.OrderReceived, 
												 Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
		if !m.ClientList[addr].Obstruction {
			m.ClientList[addr].TaskTimer.Reset(config.Request_timeout)
		}
	} else {
		client_addr := m.selectOptimalElevator(floor)
		if client_addr != "" {
			m.HallAssignments[floor][button] = m.ClientList[client_addr].ID
			m.ClientList[client_addr].ActiveReq++
			m.ClientList[client_addr].Send(network.Message{Header: network.OrderReceived, 
															Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
			if !m.ClientList[client_addr].Obstruction {
				m.ClientList[client_addr].TaskTimer.Reset(config.Request_timeout)
			}
		}
	}
	m.SendLightUpdate()
}

func (m *Master) RedistributeHallRequest(id string) {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
			if m.HallAssignments[f][b] == id {
				m.DistributeRequest(f, b, "")
			}
		}
	}
}

func (m *Master) ResendCabRequest(addr string) {
	id := m.ClientList[addr].ID
	for f := 0; f < config.N_floors; f++ {
		if m.CabRequests[f][id] {
			m.DistributeRequest(f, elevio.BT_Cab, addr)
		}
	}
}

func (m *Master) ResendHallRequest() {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
			if m.HallRequests[f][b] && m.HallAssignments[f][b] == "" {
				m.DistributeRequest(f, b, "")
			}
		}
	}
}

func (m *Master) SendLightUpdate() {
	for _, elev := range m.ClientList {
		light := utilities.NewRequests(config.N_floors, config.N_buttons)
		for f := 0; f < config.N_floors; f++ {
			copy(light[f], m.HallRequests[f])
			light[f][elevio.BT_Cab] = m.CabRequests[f][elev.ID]
		}
		elev.Send(network.Message{Header: network.LightUpdate, 
								  Payload: &network.MessagePayload{Lights: light}})
	}
}