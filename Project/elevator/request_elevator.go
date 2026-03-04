package elevator

import (
	"project/config"
	"project/elevio"
	"project/network"
)

func (e *Elevator) requestPending() bool {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) requestAbove() bool {
	for f := e.CurrentFloor + 1; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) requestBelow() bool {
	for f := 0; f < e.CurrentFloor; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) requestHere() bool {
	for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
		if e.Requests[e.CurrentFloor][b] {
			return true
		}
	}
	return false
}

func (e *Elevator) RemoveHallRequests() {
	for f := 0; f < config.N_floors; f++ {
		e.Requests[f][elevio.BT_HallUp] = false
		e.Requests[f][elevio.BT_HallDown] = false
	}
}

func (e *Elevator) shouldStop() bool {
	switch e.Direction {
	case elevio.MD_Down:
		return e.Requests[e.CurrentFloor][elevio.BT_HallDown] || e.Requests[e.CurrentFloor][elevio.BT_Cab] || !e.requestBelow()

	case elevio.MD_Up:
		return e.Requests[e.CurrentFloor][elevio.BT_HallUp] || e.Requests[e.CurrentFloor][elevio.BT_Cab] || !e.requestAbove()

	case elevio.MD_Stop:
		return !e.requestBelow() && !e.requestAbove()
	
	default:
		return false
	}
}

func (e *Elevator) shouldClear() bool {
	return ((e.Direction == elevio.MD_Up && e.Requests[e.CurrentFloor][elevio.BT_HallUp]) ||
		    (e.Direction == elevio.MD_Down && e.Requests[e.CurrentFloor][elevio.BT_HallDown]) ||
		    e.Direction == elevio.MD_Stop ||
		    e.Requests[e.CurrentFloor][elevio.BT_Cab])
}

func (e *Elevator) clearRequest(floor int, button elevio.ButtonType) {
	if !e.Requests[floor][button] {
		return
	}
	e.Requests[floor][button] = false
	if e.IsConnected {
		e.Send(network.Message{Header: network.OrderFulfilled, 
							   Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
	}
}

func (e *Elevator) clearAtCurrentFloor() {
	e.clearRequest(e.CurrentFloor, elevio.BT_Cab)

	if (e.requestAbove() && (e.Requests[e.CurrentFloor][elevio.BT_HallUp] && e.Requests[e.CurrentFloor][elevio.BT_HallDown])) {
		e.clearRequest(e.CurrentFloor, elevio.BT_HallUp)
	} else if (e.requestBelow() && (e.Requests[e.CurrentFloor][elevio.BT_HallUp] && e.Requests[e.CurrentFloor][elevio.BT_HallDown])) {
		e.clearRequest(e.CurrentFloor, elevio.BT_HallDown)
	} else {
		e.clearRequest(e.CurrentFloor, elevio.BT_HallUp)
		e.clearRequest(e.CurrentFloor, elevio.BT_HallDown)
	}
}