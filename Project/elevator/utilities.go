package elevator

import (
	"config"
	"elevio"
)

func (e *Elevator) update_lights() {
	for floor := 0; floor < config.N_floors; floor++ {
		for btn := elevio.ButtonType(0); btn < config.N_buttons; btn++ {
			elevio.SetButtonLamp(btn, floor, e.requests[floor][btn])
		}
	}
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