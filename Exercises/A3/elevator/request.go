package elevator

import (
	"config"
	"elevio"
)

func create_request_arr(rows, cols int) [][]bool {
	q := make([][]bool, rows)

	for i := range q {
		q[i] = make([]bool, cols)
	}

	return q
}

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

func (e *Elevator) clear_at_current_floor() {
	e.requests[e.current_floor][elevio.BT_Cab] = false
	
	if (e.request_above() && (e.requests[e.current_floor][elevio.BT_HallUp] && e.requests[e.current_floor][elevio.BT_HallDown])) {
		e.requests[e.current_floor][elevio.BT_HallUp] = false
	} else if (e.request_below() && (e.requests[e.current_floor][elevio.BT_HallUp] && e.requests[e.current_floor][elevio.BT_HallDown])) {
		e.requests[e.current_floor][elevio.BT_HallDown] = false
	} else {
		e.requests[e.current_floor][elevio.BT_HallUp] = false
		e.requests[e.current_floor][elevio.BT_HallDown] = false
	}
}