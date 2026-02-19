package elevator

import (
	"project/config"
	"project/elevio"
	"project/network"
	"project/utilities"
	"time"
)

type state int

const (
	Undefined state = iota
	Idle
	Moving
	Door_open
)

type Elevator struct {
	Current_state state
	Current_floor int
	Direction     elevio.MotorDirection
	Obstruction   bool
	Requests      [][]bool
	Id            string
	Successor     bool
	//Network
	Connection    *network.Client
	Connected     bool
	Retry_counter int
	Pending       map[string]*network.Message
	//Timers
	Door_timer      *time.Timer
	Door_timer_done bool
	Reconnect_timer *time.Timer
	Pending_ticker  *time.Ticker
}

func New_elevator(id string) *Elevator {
	elevator := &Elevator{
		Current_state:   Undefined,
		Current_floor:   -1,
		Requests:        utilities.Create_request_arr(config.N_floors, config.N_buttons),
		Id:              id,
		Successor:       true,
		Connected:       false,
		Retry_counter:   0,
		Pending:         make(map[string]*network.Message),
		Door_timer:      time.NewTimer(config.Open_duration),
		Reconnect_timer: time.NewTimer(config.Reconnect_delay),
		Pending_ticker:  time.NewTicker(config.Pending_resend_rate),
	}
	elevator.Door_timer.Stop()
	elevator.Door_timer_done = false

	return elevator
}
