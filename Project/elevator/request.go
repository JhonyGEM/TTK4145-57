package elevator

import (
	"config"
	"elevio"
	"network"
)

func (e *Elevator) pending_request() bool {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_above() bool {
	for f := e.Current_floor + 1; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_below() bool {
	for f := 0; f < e.Current_floor; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.Requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_here() bool {
	for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
		if e.Requests[e.Current_floor][b] {
			return true
		}
	}
	return false
}

func (e *Elevator) Remove_hall_requests() {
	for f := 0; f < config.N_floors; f++ {
		e.Requests[f][elevio.BT_HallUp] = false
		e.Requests[f][elevio.BT_HallDown] = false
	}
}

func (e *Elevator) should_stop() bool {
	switch e.Direction {
	case elevio.MD_Down:
		return e.Requests[e.Current_floor][elevio.BT_HallDown] || e.Requests[e.Current_floor][elevio.BT_Cab] || !e.request_below()

	case elevio.MD_Up:
		return e.Requests[e.Current_floor][elevio.BT_HallUp] || e.Requests[e.Current_floor][elevio.BT_Cab] || !e.request_above()
	
	default:
		return false
	}
}

func (e *Elevator) should_clear() bool {
	return ((e.Direction == elevio.MD_Up && e.Requests[e.Current_floor][elevio.BT_HallUp]) ||
		    (e.Direction == elevio.MD_Down && e.Requests[e.Current_floor][elevio.BT_HallDown]) ||
		    (e.Direction == elevio.MD_Stop) ||
		    (e.Requests[e.Current_floor][elevio.BT_Cab]))
}

func (e *Elevator) clear_at_current_floor() {
	e.Requests[e.Current_floor][elevio.BT_Cab] = false
	if e.Connected {
		e.Connection.Send(network.Message{Header: network.OrderFulfilled, Payload: &network.DataPayload{OrderFloor: e.Current_floor, OrderButton: elevio.BT_Cab}})
	}
	
	if (e.request_above() && (e.Requests[e.Current_floor][elevio.BT_HallUp] && e.Requests[e.Current_floor][elevio.BT_HallDown])) {
		e.Requests[e.Current_floor][elevio.BT_HallUp] = false
		if e.Connected {
			e.Connection.Send(network.Message{Header: network.OrderFulfilled, Payload: &network.DataPayload{OrderFloor: e.Current_floor, OrderButton: elevio.BT_HallUp}})
		}
	} else if (e.request_below() && (e.Requests[e.Current_floor][elevio.BT_HallUp] && e.Requests[e.Current_floor][elevio.BT_HallDown])) {
		e.Requests[e.Current_floor][elevio.BT_HallDown] = false
		if e.Connected {
			e.Connection.Send(network.Message{Header: network.OrderFulfilled, Payload: &network.DataPayload{OrderFloor: e.Current_floor, OrderButton: elevio.BT_HallDown}})
		}
	} else {
		e.Requests[e.Current_floor][elevio.BT_HallUp] = false
		e.Requests[e.Current_floor][elevio.BT_HallDown] = false
		if e.Connected {
			e.Connection.Send(network.Message{Header: network.OrderFulfilled, Payload: &network.DataPayload{OrderFloor: e.Current_floor, OrderButton: elevio.BT_HallUp}})
			e.Connection.Send(network.Message{Header: network.OrderFulfilled, Payload: &network.DataPayload{OrderFloor: e.Current_floor, OrderButton: elevio.BT_HallDown}})
		}
	}
}