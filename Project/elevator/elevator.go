package elevator

import (
	"project/config"
	"project/elevio"
	"project/network"
	"project/utilities"
	"project/master"
	"time"
	"log"
)

type ElevatorState int

const (
	Undefined ElevatorState = iota
	Idle
	Moving
	Door_open
)

type Elevator struct {
	Current_state   ElevatorState
	Current_floor   int
	Direction       elevio.MotorDirection
	Obstruction     bool
	Requests        [][]bool
	Id              string
	Succesor        bool
	//Network
	Connection      *network.Client
	Connected       bool
	Retry_counter   int
	Pending         map[string]*network.Message
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
		Succesor:        true,
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

func (e *Elevator) Handle_message(message network.Message, backup *master.Backup) {
	log.Printf("Recieved: %v", message.Header)
	switch message.Header {
	case network.OrderReceived:
		e.Requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
		e.Save_pending()
		e.Step_FSM()

	case network.LightUpdate:
		e.Update_lights(message.Payload.Lights)

	case network.Backup:
		log.Printf("Recieved backup")
		backup.Cab_reg = message.Payload.BackupCab
		backup.Hall_reg = message.Payload.BackupHall
		e.Send(network.Message{Header: network.Ack, UID: message.UID})

	case network.Ack:
		_, ok := e.Pending[message.UID]
		if ok {
			delete(e.Pending, message.UID)
			e.Save_pending()
		}

	case network.Succesor:
		e.Succesor = true
	}
}