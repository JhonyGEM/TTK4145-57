package elevator

import (
	"project/config"
	"project/elevio"
)

func (e *Elevator) StepFSM() {
	switch e.Local.State {
	case Undefined:
		e.Local.SetButtonLights()
		e.Local.Obstruction = elevio.GetObstruction()
		elevio.SetDoorOpenLamp(false)

		if elevio.GetFloor() == -1 {
			e.Local.updateDirection(elevio.MD_Down)
		} else {
			e.Local.CurrentFloor = elevio.GetFloor()
			elevio.SetFloorIndicator(e.Local.CurrentFloor)
			e.Local.updateDirection(elevio.MD_Stop)
			if e.Local.Obstruction {
				e.UpdateState(DoorOpen)
			} else {
				e.UpdateState(Idle)
			}
		}

	case Idle:
		if e.Local.RequestPending() {
			if e.Local.requestHere() {
				e.clearAtCurrentFloor()
				e.Local.SetButtonLights()
				e.Timers.Door.Reset(config.Open_duration)
				e.UpdateState(DoorOpen)
			} else {
				e.UpdateState(Moving)
			}
		}

	case Moving:
		if (e.Local.shouldStop() && elevio.GetFloor() != -1) {
			e.Local.updateDirection(elevio.MD_Stop)
			if e.Local.shouldClear() {
				e.clearAtCurrentFloor()
				e.Local.SetButtonLights()
			}
			e.Timers.Door.Reset(config.Open_duration)	
			e.UpdateState(DoorOpen)
		} else {
			e.Local.chooseDirection()
		}

	case DoorOpen:
		elevio.SetDoorOpenLamp(true)

		if e.Local.Obstruction {
			e.Timers.Door.Reset(config.Open_duration)
			e.Local.DoorTimerDone = false
		} else {
			if e.Local.DoorTimerDone {
				elevio.SetDoorOpenLamp(false)
				e.Local.DoorTimerDone = false
				if e.Local.RequestPending() {
					e.UpdateState(Moving)
				} else {
					e.UpdateState(Idle)
				}
			}
		}
	}
}

func (e *Elevator) UpdateState(newState ElevatorState) {
	e.Local.State = newState
	e.StepFSM()
}

func (ls *LocalState) updateDirection(newDirection elevio.MotorDirection) {
	ls.Direction = newDirection
	elevio.SetMotorDirection(newDirection)
}

func (ls *LocalState) chooseDirection() {
	switch ls.Direction {
	case elevio.MD_Up:
		if ls.requestAbove() {
			ls.Direction = elevio.MD_Up
		} else if ls.requestBelow() {
			ls.Direction = elevio.MD_Down
		} else {
			ls.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(ls.Direction)
	
	case elevio.MD_Down:
		if ls.requestBelow() {
			ls.Direction = elevio.MD_Down
		} else if ls.requestAbove() {
			ls.Direction = elevio.MD_Up
		} else {
			ls.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(ls.Direction)

	case elevio.MD_Stop:
		if ls.requestAbove() {
			ls.Direction = elevio.MD_Up
		} else if ls.requestBelow() {
			ls.Direction = elevio.MD_Down
		} else {
			ls.Direction = elevio.MD_Stop
		}
		elevio.SetMotorDirection(ls.Direction)
	}
}