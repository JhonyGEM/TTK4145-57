package mainFunctions

import (
	"project/config"
	"project/elevator"
	"project/elevio"
	"project/master"
	"project/network"
	"project/utilities"
)

func FloorHandler(floor int, elevator *elevator.Elevator) {
	if floor != -1 {
		elevator.Current_floor = floor
		elevio.SetFloorIndicator(floor)
		elevator.Step_FSM()
		if elevator.Connected {
			elevator.Connection.Send(network.Message{
				Header: network.FloorUpdate,
				Payload: &network.DataPayload{
					CurrentFloor: elevator.Current_floor,
				},
			})
		}
	}
}

func ButtonHandler(button elevio.ButtonEvent, previousButton elevio.ButtonEvent, elevator *elevator.Elevator) {
	if previousButton != button {
		previousButton = button
		if elevator.Connected {
			message := network.Message{
				Header: network.OrderReceived,
				Payload: &network.DataPayload{
					OrderFloor:  button.Floor,
					OrderButton: button.Button,
				},
				UID: utilities.Gen_uid(elevator.Id),
			}
			elevator.Pending[message.UID] = &message
			elevator.Send(message)
		} else {
			if button.Button == elevio.BT_Cab {
				elevator.Requests[button.Floor][button.Button] = true
				elevator.Update_lights(elevator.Requests)
				elevator.Step_FSM()
			}
		}
	}
}

func ObstructionHandler(obstruction bool, elevator *elevator.Elevator) {
	elevator.Obstruction = obstruction
	elevator.Step_FSM()
	if elevator.Connected {
		elevator.Connection.Send(network.Message{
			Header: network.ObstructionUpdate,
			Payload: &network.DataPayload{
				Obstruction: elevator.Obstruction,
			},
		})
	}
}

func DoorTimerHandler(elevator *elevator.Elevator) {
	elevator.Door_timer.Stop()
	elevator.Door_timer_done = true
	elevator.Step_FSM()
}

func LossConnectionHandler(elev *elevator.Elevator) {
	elev.Connected = false
	elev.Connection = &network.Client{}
	elev.Reconnect_timer.Reset(config.Reconnect_delay)
	elev.Remove_hall_requests()
	elev.Update_state(elevator.Undefined)
	elev.Step_FSM()
}

func AddNewClient(m *master.Master, conn *network.Client) {
	if len(m.Client_list) == 0 {
		m.Successor_addr = conn.Addr
	}
	m.Add_client(conn)
}

func RemoveClient(m *master.Master, conn *network.Client) {
	id := m.Client_list[conn.Addr].ID
	m.Remove_client(conn.Addr)

	if len(m.Client_list) == 0 {
		panic("Loss of connectivity")
	} else {
		if conn.Addr == m.Successor_addr {
			for _, c := range m.Client_list {
				m.Successor_addr = c.Connection.Addr
				break
			}
		}

		m.Redistribute_request(id)
	}
}
