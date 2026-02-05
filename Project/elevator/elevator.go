package elevator

import (
	"config"
	"elevio"
	"network"
	"time"
	"utilities"
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
	succesor        bool
	//Network
	Connection      *network.Client
	connected       bool
	retry_counter   int
	//Timers
	door_timer      *time.Timer
	door_timer_done bool
	reconnect_timer *time.Timer
}

func new_elevator(id string) *Elevator {
	elevator := &Elevator{
		current_state:   undefined,
		current_floor:   -1,
		requests:        utilities.Create_request_arr(config.N_floors, config.N_buttons),
		id:              id,
		succesor:        true,
		connected:       false,
		retry_counter:   0,
		door_timer:      time.NewTimer(config.Open_duration),
		reconnect_timer: time.NewTimer(config.Reconnect_delay),
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

	lossChan := make(chan *network.Client)
	msgChan := make(chan network.Message)
	var prev_btn elevio.ButtonEvent

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
				if elevator.connected {
					elevator.Connection.Send(network.FloorUpdate, &network.DataPayload{CurrentFloor: elevator.current_floor})
				}
			}
		
		case btn := <-drv_buttons:
			if prev_btn != btn {
				prev_btn = btn
				if elevator.connected {
					elevator.Connection.Send(network.OrderReceived, &network.DataPayload{OrderFloor: btn.Floor, OrderButton: btn.Button})
				} else {
					if btn.Button == elevio.BT_Cab {
						elevator.requests[btn.Floor][btn.Button] = true
						update_lights(elevator.requests)
						elevator.step_FSM()
					} 
				}
			}

		case obs := <-drv_obstruction:
			elevator.obstruction = obs
			elevator.step_FSM()
			if elevator.connected {
				elevator.Connection.Send(network.ObstructionUpdate, &network.DataPayload{Obstruction: elevator.obstruction})
			}

		case <-elevator.door_timer.C:
			elevator.door_timer.Stop()
			elevator.door_timer_done = true
			elevator.step_FSM()

		case <-elevator.reconnect_timer.C:
			elevator.reconnect_timer.Stop()

			if elevator.retry_counter > 3 && elevator.succesor {
				utilities.Start_new_master()
				continue
			}

			addr, err := network.Discover_server()
			if err != nil {
				elevator.reconnect_timer.Reset(config.Reconnect_delay)
				elevator.retry_counter++
				continue
			}
			conn, err := network.Connect(addr)
			if err != nil {
				elevator.reconnect_timer.Reset(config.Reconnect_delay)
				elevator.retry_counter++
				continue
			}

			elevator.retry_counter = 0 
			elevator.Connection = network.New_client(conn)
			elevator.connected = true
			elevator.Connection.Send(network.ClientInfo, &network.DataPayload{ID: elevator.id, CurrentFloor: elevator.current_floor, Obstruction: elevator.obstruction})
			go elevator.Connection.Listen(msgChan, lossChan)
			go elevator.Connection.Heart_beat()

		case <-lossChan:
			elevator.connected = false
			elevator.Connection = &network.Client{}
			elevator.reconnect_timer.Reset(config.Reconnect_delay)
			elevator.remove_hall_requests()
			elevator.update_state(undefined)
			elevator.step_FSM()

		case message := <-msgChan:
			switch message.Header {
			case network.OrderReceived:
				elevator.requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
				update_lights(elevator.requests)
				elevator.step_FSM()

			case network.LightUpdate:
				update_lights(message.Payload.Lights)

			}
		}
	}
}