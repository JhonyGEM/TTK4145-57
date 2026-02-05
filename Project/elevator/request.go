package elevator

import (
	"config"
	"elevio"
	"network"
)

func (e *Elevator) pending_request() bool {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_above() bool {
	for f := e.current_floor + 1; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_below() bool {
	for f := 0; f < e.current_floor; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if e.requests[f][b] {
				return true
			}
		}
	}
	return false
}

func (e *Elevator) request_here() bool {
	for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
		if e.requests[e.current_floor][b] {
			return true
		}
	}
	return false
}

func (e *Elevator) remove_hall_requests() {
	for f := 0; f < config.N_floors; f++ {
		e.requests[f][elevio.BT_HallUp] = false
		e.requests[f][elevio.BT_HallDown] = false
	}
}

func (e *Elevator) should_stop() bool {
	switch e.direction {
	case elevio.MD_Down:
		return e.requests[e.current_floor][elevio.BT_HallDown] || e.requests[e.current_floor][elevio.BT_Cab] || !e.request_below()

	case elevio.MD_Up:
		return e.requests[e.current_floor][elevio.BT_HallUp] || e.requests[e.current_floor][elevio.BT_Cab] || !e.request_above()
	
	default:
		return false
	}
}

func (e *Elevator) should_clear() bool {
	return ((e.direction == elevio.MD_Up && e.requests[e.current_floor][elevio.BT_HallUp]) ||
		    (e.direction == elevio.MD_Down && e.requests[e.current_floor][elevio.BT_HallDown]) ||
		    (e.direction == elevio.MD_Stop) ||
		    (e.requests[e.current_floor][elevio.BT_Cab]))
}

// TODO: need testing
func (e *Elevator) clear_at_current_floor() {
	e.requests[e.current_floor][elevio.BT_Cab] = false
	if e.connected {
		e.Connection.Send(network.OrderFulfilled, &network.DataPayload{OrderFloor: e.current_floor, OrderButton: elevio.BT_Cab})
	}
	
	if (e.request_above() && (e.requests[e.current_floor][elevio.BT_HallUp] && e.requests[e.current_floor][elevio.BT_HallDown])) {
		e.requests[e.current_floor][elevio.BT_HallUp] = false
		if e.connected {
			e.Connection.Send(network.OrderFulfilled, &network.DataPayload{OrderFloor: e.current_floor, OrderButton: elevio.BT_HallUp})
		}
	} else if (e.request_below() && (e.requests[e.current_floor][elevio.BT_HallUp] && e.requests[e.current_floor][elevio.BT_HallDown])) {
		e.requests[e.current_floor][elevio.BT_HallDown] = false
		if e.connected {
			e.Connection.Send(network.OrderFulfilled, &network.DataPayload{OrderFloor: e.current_floor, OrderButton: elevio.BT_HallDown})
		}
	} else {
		e.requests[e.current_floor][elevio.BT_HallUp] = false
		e.requests[e.current_floor][elevio.BT_HallDown] = false
		if e.connected {
			e.Connection.Send(network.OrderFulfilled, &network.DataPayload{OrderFloor: e.current_floor, OrderButton: elevio.BT_HallUp})
			e.Connection.Send(network.OrderFulfilled, &network.DataPayload{OrderFloor: e.current_floor, OrderButton: elevio.BT_HallDown})
		}
	}
}