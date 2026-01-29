package elevator

import (
	"config"
	"elevio"
	"time"
)

type state int

const (
	undefined state = iota
	idle
	moving
	door_open
)

type Elevator struct {
	current_state   state
	current_floor   int
	direction       elevio.MotorDirection
	obstruction     bool
	requests        [][]bool
	id              string
	//Timers
	door_timer      *time.Timer
	door_timer_done bool
}

func new_elevator(id string) *Elevator {
	elevator := &Elevator{
		current_state:   undefined,
		current_floor:    -1,
		requests:        create_request_arr(config.N_floors, config.N_buttons),
		id:              id,
		door_timer:      time.NewTimer(config.Door_open_duration),
	}
	elevator.door_timer.Stop()
	elevator.door_timer_done = false

	return elevator
}

func Run_elevator(id string) {
	elevator := new_elevator(id)
	elevio.Init("localhost:15657", config.N_floors)
	
	drv_buttons := make(chan elevio.ButtonEvent, config.N_floors * config.N_buttons)
	drv_floors := make(chan int)
	drv_obstruction := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstruction)

	elevator.step_FSM()

	for {
		select {
		case floor := <-drv_floors:
			if floor != -1 {
				elevator.current_floor = floor
				elevio.SetFloorIndicator(floor)
				elevator.step_FSM()
			}

		case btn := <-drv_buttons:
			if !(elevator.requests[btn.Floor][btn.Button]) {
				elevator.requests[btn.Floor][btn.Button] = true
				elevator.update_lights()
				elevator.step_FSM()
			}

		case obs := <-drv_obstruction:
			elevator.obstruction = obs
			elevator.step_FSM()

		case <-elevator.door_timer.C:
			elevator.door_timer.Stop()
			elevator.door_timer_done = true
			elevator.step_FSM()
		}
	}
}