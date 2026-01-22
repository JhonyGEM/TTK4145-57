package elevator

import (
	"config"
	"elevio"
	"time"
)

var state = struct {
	undefined, idle, moving, door_open string
}{
	"undefined", "idle", "moving", "door_open",
}

type Elevator struct {
	current_state   string
	current_floor   int
	direction       elevio.MotorDirection
	obstruction     bool
	requests        [][]bool
	id              string
	//Timers
	door_timer      *time.Timer
	door_timer_done bool
}

func create_elevator(id string) *Elevator {
	Elevator := &Elevator{
		current_state:   state.undefined,
		current_floor:    -1,
		requests:        create_request_arr(config.N_floors, config.N_buttons),
		id:              id,
		door_timer:      time.NewTimer(config.Door_open_duration),
	}
	Elevator.door_timer.Stop()
	Elevator.door_timer_done = false

	return Elevator
}

func (e *Elevator) update_lights() {
	for floor := 0; floor < config.N_floors; floor++ {
		for btn := elevio.ButtonType(0); btn < config.N_buttons; btn++ {
			elevio.SetButtonLamp(btn, floor, e.requests[floor][btn])
		}
	}
}

func (e *Elevator) state_machine() {
	for {
		switch e.current_state {
		case state.undefined:
			e.update_lights()
			elevio.SetDoorOpenLamp(false)

			if e.current_floor == -1 {
				e.direction = elevio.MD_Down
				elevio.SetMotorDirection(e.direction)
			} else {
				e.direction = elevio.MD_Stop
				elevio.SetMotorDirection(e.direction)
				e.current_state = state.idle
			}

		case state.idle:
			if (e.pending_request()) {
				if (e.request_here()) {
					e.door_timer.Reset(config.Door_open_duration)
					e.current_state = state.door_open
				} else {
					e.current_state = state.moving
				}
			}

		case state.moving:
			if (e.should_stop() && elevio.GetFloor() != -1) {
				e.direction = elevio.MD_Stop
				elevio.SetMotorDirection(e.direction)

				e.door_timer.Reset(config.Door_open_duration)
				e.current_state = state.door_open
			} else {
				e.choose_direction()
				elevio.SetMotorDirection(e.direction)
			}

		case state.door_open:
			elevio.SetDoorOpenLamp(true)
			if e.should_clear() {
				e.clear_at_current_floor()
			}
			e.update_lights()
			if (e.obstruction) {
				e.door_timer.Reset(config.Door_open_duration)
			} else {
				if e.door_timer_done {
					elevio.SetDoorOpenLamp(false)
					e.door_timer_done = false
					if e.pending_request() {
						e.current_state = state.moving
					} else {
						e.current_state = state.idle
					}
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}


func Run_elevator(id string) {
	elevator := create_elevator(id)

	elevio.Init("localhost:15657", config.N_floors)
	
	drv_buttons := make(chan elevio.ButtonEvent, config.N_floors*3)
	drv_floors := make(chan int)
	drv_obstruction := make(chan bool)

	go elevio.PollButtons(drv_buttons)
	go elevio.PollFloorSensor(drv_floors)
	go elevio.PollObstructionSwitch(drv_obstruction)

	go elevator.state_machine()

	for {
		select {
		case floor := <-drv_floors:
			if floor != -1 {
				elevator.current_floor = floor
				elevio.SetFloorIndicator(floor)
			}

		case btn := <-drv_buttons:
			if !(elevator.requests[btn.Floor][btn.Button]) {
				elevator.requests[btn.Floor][btn.Button] = true
				elevator.update_lights()
			}

		case obs := <-drv_obstruction:
			elevator.obstruction = obs

		case <-elevator.door_timer.C:
			elevator.door_timer.Stop()
			elevator.door_timer_done = true
		}
	}
}