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
	DoorOpen
)

type Elevator struct {
	CurrentState    ElevatorState
	CurrentFloor    int
	Direction       elevio.MotorDirection
	Obstruction     bool
	Requests        [][]bool
	ID              string
	IsSuccessor      bool
	//Network
	Connection      *network.Client
	IsConnected     bool
	RetryCounter    int
	Sequence        int
	Pending         map[string]*network.Pending    // key = uid
	//Timers
	DoorTimer       *time.Timer
	DoorTimerDone   bool
	ReconnectTimer  *time.Timer
	PendingTicker   *time.Ticker
}

func NewElevator(id string, successor bool) *Elevator {
	elevator := &Elevator{
		CurrentState:    Undefined,
		CurrentFloor:   -1,
		Requests:        utilities.NewRequests(config.N_floors, config.N_buttons),
		ID:              id,
		IsSuccessor:      successor,
		Pending:         make(map[string]*network.Pending),
		DoorTimer:       time.NewTimer(config.Open_duration),
		ReconnectTimer:  time.NewTimer(config.Reconnect_delay),
		PendingTicker:   time.NewTicker(config.Pending_check_rate),
	}
	elevator.DoorTimer.Stop()
	elevator.DoorTimerDone = false

	return elevator
}

func (e *Elevator) HandleMessage(message network.Message, backup *master.Backup) {
	switch message.Header {
	case network.OrderReceived:
		log.Printf("Recieved order: Floor %d, %v", message.Payload.OrderFloor, message.Payload.OrderButton)
		e.Requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
		e.StepFSM()

	case network.LightUpdate:
		e.UpdateLights(message.Payload.Lights)

	case network.Backup:
		log.Printf("Recieved: %v", message.Header)
		backup.CabRequests = message.Payload.BackupCab
		backup.HallRequests = message.Payload.BackupHall
		e.Send(network.Message{Header: network.Ack, UID: message.UID})

	case network.Ack:
		_, ok := e.Pending[message.UID]
		if ok {
			delete(e.Pending, message.UID)
			utilities.SaveToFile(config.Pending_backup, e.Pending)
		}

	case network.Successor:
		log.Println("Elevator is now the successor")
		e.IsSuccessor = true
		e.Send(network.Message{Header: network.Ack, UID: message.UID})
	}
}