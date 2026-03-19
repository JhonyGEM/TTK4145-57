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
	Local			LocalState
	Network			NetworkState
	Timers			ElevatorTimers
}

type LocalState struct {
	State    		ElevatorState
	CurrentFloor   	int
	Direction       elevio.MotorDirection
	Obstruction     bool
	Requests        [][]bool						// [floor][button]
	Lights			[][]bool						// [floor][button]
	DoorTimerDone	bool
}

type NetworkState struct {
	ID              string	
	IsSuccessor		bool
	Connection      *network.Client
	IsConnected     bool
	RetryCounter    int
	Sequence        int
	Pending         map[string]*network.Pending		// [uid]
}

type ElevatorTimers struct {
	Door			*time.Timer
	Reconnect		*time.Timer
	PendingTicker	*time.Ticker
}

func NewElevator(id string, successor bool) *Elevator {
	elevator := &Elevator{
		Local: LocalState{
			State:			Undefined, 
			CurrentFloor:	-1, 
			Requests:		utilities.NewRequests(), 
			Lights:			utilities.NewRequests()},
		Network: NetworkState{
			ID:				id, 
			IsSuccessor: 	successor, 
			Pending: 		make(map[string]*network.Pending)},
		Timers: ElevatorTimers{
			Door: 			time.NewTimer(config.Open_duration), 
			Reconnect: 		time.NewTimer(config.Reconnect_delay), 
			PendingTicker: 	time.NewTicker(config.Pending_check_rate)},
	}
	elevator.Timers.Door.Stop()
	elevator.Local.DoorTimerDone = false

	return elevator
}

func (e *Elevator) HandleMessage(message network.Message, backup *master.Backup) {
	switch message.Header {
	case network.OrderReceived:
		log.Printf("Received order: Floor %d, %v", message.Payload.OrderFloor, message.Payload.OrderButton)
		e.Local.Requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
		e.StepFSM()

	case network.LightUpdate:
		e.Local.Lights = message.Payload.Lights
		e.Local.SetButtonLights()

	case network.Backup:
		log.Printf("Received: %v", message.Header)
		backup.CabRequests = message.Payload.BackupCab
		backup.HallRequests = message.Payload.BackupHall
		e.Network.Send(network.Message{
			Header: network.Ack, 
			UID: message.UID})

	case network.Ack:
		_, ok := e.Network.Pending[message.UID]
		if ok {
			delete(e.Network.Pending, message.UID)
			utilities.SaveToFile(config.Pending_backup, e.Network.Pending)
		}

	case network.Successor:
		log.Println("Elevator is now the successor")
		e.Network.IsSuccessor = true
		e.Network.Send(network.Message{
			Header: network.Ack, 
			UID: message.UID})

	case network.NotSuccessor:
		e.Network.IsSuccessor = false
	}
}

func (e *Elevator) HandleFloorUpdate(floor int) {
	if floor != -1 {
		e.Local.CurrentFloor = floor
		elevio.SetFloorIndicator(floor)
		e.StepFSM()
		if e.Network.IsConnected {
			e.Network.Send(network.Message{
				Header: network.FloorUpdate,
				Payload: &network.MessagePayload{CurrentFloor: e.Local.CurrentFloor}})
		}
	}
}

func (e *Elevator) HandleButtonPress(btn elevio.ButtonEvent) {
	if e.Network.IsConnected {
		message := network.Message{
			Header: network.OrderReceived,
			Payload: &network.MessagePayload{OrderFloor: btn.Floor, OrderButton: btn.Button},
			UID: utilities.GenUID(e.Network.ID, e.Network.Sequence)}
		e.Network.Sequence++				
		e.Network.Pending[message.UID] = &network.Pending{Message: message, Timestamp: time.Now()}
		utilities.SaveToFile(config.Pending_backup, e.Network.Pending)
		e.Network.Send(message)
	} else {
		if btn.Button == elevio.BT_Cab {
			e.Local.Requests[btn.Floor][btn.Button] = true
			e.Local.SetButtonLights()
			e.StepFSM()
		}
	}
}

func (e *Elevator) HandleObstructionUpdate(obstruction bool) {
	e.Local.Obstruction = obstruction
	if !e.Local.Obstruction && e.Timers.Door.Stop() {
		e.Timers.Door.Reset(config.Open_duration)
	}
	e.StepFSM()
	if e.Network.IsConnected {
		e.Network.Send(network.Message{
			Header: network.ObstructionUpdate,
			Payload: &network.MessagePayload{Obstruction: e.Local.Obstruction}})
	}
}

func (e *Elevator) HandleDisconnect() {
	log.Println("Disconnected from server")
	e.Network.IsConnected = false
	e.Network.Connection = &network.Client{}
	e.Timers.Reconnect.Reset(config.Reconnect_delay)
	e.Local.clearHallRequests()
	e.Local.Lights = e.Local.Requests
	e.Local.SetButtonLights()
	if elevio.GetFloor() == -1 && !e.Local.RequestPending() {
		e.UpdateState(Undefined)
	}
	e.StepFSM()
}