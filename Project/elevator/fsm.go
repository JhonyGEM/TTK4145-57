package elevator

import (
	"project/config"
	"project/elevio"
)

func (e *Elevator) StepFSM() {
	switch e.CurrentState {
	case Undefined:
		e.UpdateLights(e.Requests)
		elevio.SetDoorOpenLamp(false)

		if elevio.GetFloor() == -1 {
			e.updateDirection(elevio.MD_Down)
		} else {
			e.updateDirection(elevio.MD_Stop)
			e.UpdateState(Idle)
		}

	case Idle:
		if e.requestPending() {
			if e.requestHere() {
				e.clearAtCurrentFloor()
				if !e.IsConnected {
					e.UpdateLights(e.Requests)
				}
				e.DoorTimer.Reset(config.Open_duration)
				e.UpdateState(DoorOpen)
			} else {
				e.UpdateState(Moving)
			}
		}

	case Moving:
		if (e.shouldStop() && elevio.GetFloor() != -1) {
			e.updateDirection(elevio.MD_Stop)
			if e.shouldClear() {
				e.clearAtCurrentFloor()
				if !e.IsConnected {
					e.UpdateLights(e.Requests)
				}
			}
			e.DoorTimer.Reset(config.Open_duration)	
			e.UpdateState(DoorOpen)
		} else {
			e.chooseDirection()
		}

	case DoorOpen:
		elevio.SetDoorOpenLamp(true)

		if e.Obstruction {
			e.DoorTimer.Reset(config.Open_duration)
			e.DoorTimerDone = false
		} else {
			if e.DoorTimerDone {
				elevio.SetDoorOpenLamp(false)
				e.DoorTimerDone = false
				if e.requestPending() {
					e.UpdateState(Moving)
				} else {
					e.UpdateState(Idle)
				}
			}
		}
	}
}

func (e *Elevator) UpdateState(newState ElevatorState) {
	e.CurrentState = newState
	e.StepFSM()
}

func (e *Elevator) updateDirection(newDirection elevio.MotorDirection) {
	e.Direction = newDirection
	elevio.SetMotorDirection(newDirection)
}

func (e *Elevator) chooseDirection() {
	switch e.Direction {
	case elevio.MD_Up:
		if e.requestAbove() {
			e.Direction = elevio.MD_Up
		} else if e.requestBelow() {
			e.Direction = elevio.MD_Down
		} else {
			e.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(e.Direction)
	
	case elevio.MD_Down:
		if e.requestBelow() {
			e.Direction = elevio.MD_Down
		} else if e.requestAbove() {
			e.Direction = elevio.MD_Up
		} else {
			e.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(e.Direction)

	case elevio.MD_Stop:
		if e.requestAbove() {
			e.Direction = elevio.MD_Up
		} else if e.requestBelow() {
			e.Direction = elevio.MD_Down
		} else {
			e.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(e.Direction)
	}
}