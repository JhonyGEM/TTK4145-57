package elevator

import (
	"config"
	"elevio"
)

func (e *Elevator) update_state(new_state state) {
	e.current_state = new_state
	e.step_FSM()
}

func (e *Elevator) step_FSM() {
	switch e.current_state {
	case undefined:
		e.update_lights()
		elevio.SetDoorOpenLamp(false)

		if e.current_floor == -1 {
			e.update_direction(elevio.MD_Down)
		} else {
			e.update_direction(elevio.MD_Stop)
			e.update_state(idle)
		}

	case idle:
		if (e.pending_request()) {
			if (e.request_here()) {
				e.door_timer.Reset(config.Door_open_duration)
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
				e.update_lights()
			}
			e.door_timer.Reset(config.Door_open_duration)	
			e.update_state(door_open)
		} else {
			e.choose_direction()
			e.update_direction(e.direction)
		}

	case door_open:
		elevio.SetDoorOpenLamp(true)

		if (e.obstruction) {
			e.door_timer.Reset(config.Door_open_duration)
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