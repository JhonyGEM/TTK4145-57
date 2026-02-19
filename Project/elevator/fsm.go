package elevator

import (
	"project/config"
	"project/elevio"
)

func (e *Elevator) Step_FSM() {
	switch e.Current_state {
	case Undefined:
		e.Update_lights(e.Requests)
		elevio.SetDoorOpenLamp(false)

		if elevio.GetFloor() == -1 {
			e.update_direction(elevio.MD_Down)
		} else {
			e.update_direction(elevio.MD_Stop)
			e.Update_state(Idle)
		}

	case Idle:
		if e.pending_request() {
			if e.request_here() {
				e.clear_at_current_floor()
				if !e.Connected {
					e.Update_lights(e.Requests)
				}
				e.Door_timer.Reset(config.Open_duration)
				e.Update_state(Door_open)
			} else {
				e.Update_state(Moving)
			}
		}

	case Moving:
		if e.should_stop() && elevio.GetFloor() != -1 {
			e.update_direction(elevio.MD_Stop)
			if e.should_clear() {
				e.clear_at_current_floor()
				if !e.Connected {
					e.Update_lights(e.Requests)
				}
			}
			e.Door_timer.Reset(config.Open_duration)
			e.Update_state(Door_open)
		} else {
			e.choose_direction()
			e.update_direction(e.Direction)
		}

	case Door_open:
		elevio.SetDoorOpenLamp(true)

		if e.Obstruction {
			e.Door_timer.Reset(config.Open_duration)
		} else {
			if e.Door_timer_done {
				elevio.SetDoorOpenLamp(false)
				e.Door_timer_done = false
				if e.pending_request() {
					e.Update_state(Moving)
				} else {
					e.Update_state(Idle)
				}
			}
		}
	}
}

func (e *Elevator) Update_state(new_state state) {
	e.Current_state = new_state
	e.Step_FSM()
}

func (e *Elevator) update_direction(new_direction elevio.MotorDirection) {
	e.Direction = new_direction
	elevio.SetMotorDirection(new_direction)
}

func (e *Elevator) choose_direction() {
	switch e.Direction {
	case elevio.MD_Up:
		if e.request_above() {
			e.Direction = elevio.MD_Up
		} else if e.request_below() {
			e.Direction = elevio.MD_Down
		} else {
			e.Direction = elevio.MD_Stop
		}
	
	case elevio.MD_Down:
		if e.request_below() {
			e.Direction = elevio.MD_Down
		} else if e.request_above() {
			e.Direction = elevio.MD_Up
		} else {
			e.Direction = elevio.MD_Stop
		}

	case elevio.MD_Stop:
		if e.request_above() {
			e.Direction = elevio.MD_Up
		} else if e.request_below() {
			e.Direction = elevio.MD_Down
		} else {
			e.Direction = elevio.MD_Stop
		}
	}
}