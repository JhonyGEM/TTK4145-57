package elevator

import (
	"config"
	"elevio"
)

func (e *Elevator) step_FSM() {
	switch e.current_state {
	case undefined:
		update_lights(e.requests)
		elevio.SetDoorOpenLamp(false)

		if elevio.GetFloor() == -1 {
			e.update_direction(elevio.MD_Down)
		} else {
			e.update_direction(elevio.MD_Stop)
			e.update_state(idle)
		}

	case idle:
		if (e.pending_request()) {
			if (e.request_here()) {
				e.door_timer.Reset(config.Open_duration)
				e.update_state(door_open)
			} else {
				e.update_state(moving)
			}
		}

	case moving:
		if (e.should_stop() && elevio.GetFloor() != -1) {
			e.update_direction(elevio.MD_Stop)
			if e.should_clear() {
				e.clear_at_current_floor()
				update_lights(e.requests)
			}
			e.door_timer.Reset(config.Open_duration)	
			e.update_state(door_open)
		} else {
			e.choose_direction()
			e.update_direction(e.direction)
		}

	case door_open:
		elevio.SetDoorOpenLamp(true)

		if (e.obstruction) {
			e.door_timer.Reset(config.Open_duration)
		} else {
			if e.door_timer_done {
				elevio.SetDoorOpenLamp(false)
				e.door_timer_done = false
				if e.pending_request() {
					e.update_state(moving)
				} else {
					e.update_state(idle)
				}
			}
		}
	}
}

func update_lights(request [][]bool) {
	for floor := 0; floor < config.N_floors; floor++ {
		for btn := elevio.ButtonType(0); btn < config.N_buttons; btn++ {
			elevio.SetButtonLamp(btn, floor, request[floor][btn])
		}
	}
}

func (e *Elevator) update_state(new_state state) {
	e.current_state = new_state
	e.step_FSM()
}

func (e *Elevator) update_direction(new_direction elevio.MotorDirection) {
	e.direction = new_direction
	elevio.SetMotorDirection(new_direction)
}

func (e *Elevator) choose_direction() {
	switch e.direction {
	case elevio.MD_Up:
		if e.request_above() {
			e.direction = elevio.MD_Up
		} else if e.request_below() {
			e.direction = elevio.MD_Down
		} else {
			e.direction = elevio.MD_Stop
		}
	
	case elevio.MD_Down:
		if e.request_below() {
			e.direction = elevio.MD_Down
		} else if e.request_above() {
			e.direction = elevio.MD_Up
		} else {
			e.direction = elevio.MD_Stop
		}

	case elevio.MD_Stop:
		if e.request_above() {
			e.direction = elevio.MD_Up
		} else if e.request_below() {
			e.direction = elevio.MD_Down
		} else {
			e.direction = elevio.MD_Stop
		}
	}
}