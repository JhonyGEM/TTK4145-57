package master

import (
	"project/config"
	"project/elevio"
	"project/utilities"
	"project/network"
)

func (m *Master) selectOptimalElevator(floor int) string {
	addr := ""
	lowestCost := 9999

	for _, client := range m.ClientList {
		cost := utilities.Abs(client.CurrentFloor - floor) + client.ActiveRequestCount * config.Cost_penalty
		
		if client.Obstruction {
			cost += config.Cost_penalty
		}

		if cost < lowestCost {
			addr = client.Connection.Addr
			lowestCost = cost
		}
	}
	return addr
}

func (m *Master) IsRequestActive(floor int, button elevio.ButtonType, addr string) bool {
	if button == elevio.BT_Cab {
		return m.Requests.Cab[floor][m.ClientList[addr].ID]
	} else {
		return m.Requests.Hall[floor][button]
	}
}

func (m *Master) DistributeRequest(floor int, button elevio.ButtonType, addr string) {
	if button == elevio.BT_Cab {
		m.ClientList[addr].ActiveRequestCount++
		m.ClientList[addr].Send(network.Message{
			Header: network.OrderReceived, 
			Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
		if !m.ClientList[addr].Obstruction {
			m.ClientList[addr].TaskTimer.Reset(config.Request_timeout)
		}
	} else {
		clientAddr := m.selectOptimalElevator(floor)
		if clientAddr != "" {
			m.Requests.HallAssignments[floor][button] = m.ClientList[clientAddr].ID
			m.ClientList[clientAddr].ActiveRequestCount++
			m.ClientList[clientAddr].Send(network.Message{
				Header: network.OrderReceived, 
				Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
			if !m.ClientList[clientAddr].Obstruction {
				m.ClientList[clientAddr].TaskTimer.Reset(config.Request_timeout)
			}
		}
	}
	m.SendLightUpdate()
}

func (m *Master) ResetHallAssignments(id string) {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
			if m.Requests.HallAssignments[f][b] == id {
				m.Requests.HallAssignments[f][b] = ""
			}
		}
	}
}

func (m *Master) ResendCabRequest(addr string) {
	id := m.ClientList[addr].ID
	for f := 0; f < config.N_floors; f++ {
		if m.Requests.Cab[f][id] {
			m.DistributeRequest(f, elevio.BT_Cab, addr)
		}
	}
}

func (m *Master) ResendHallRequest() {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < elevio.ButtonType(2); b++ {
			if m.Requests.Hall[f][b] && m.Requests.HallAssignments[f][b] == "" {
				m.DistributeRequest(f, b, "")
			}
		}
	}
}

func (m *Master) SendLightUpdate() {
	for _, elev := range m.ClientList {
		light := utilities.NewRequests()
		for f := 0; f < config.N_floors; f++ {
			copy(light[f], m.Requests.Hall[f])
			light[f][elevio.BT_Cab] = m.Requests.Cab[f][elev.ID]
		}
		elev.Send(network.Message{
			Header: network.LightUpdate, 
			Payload: &network.MessagePayload{Lights: light}})
	}
}