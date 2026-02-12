package elevator

import (
	"config"
	"elevio"
	"network"
	"time"
	"utilities"
	"encoding/json"
	"os"
	"log"
)

type state int

const (
	Undefined state = iota
	Idle
	Moving
	Door_open
)

type Elevator struct {
	Current_state   state
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
		Pending_ticker:  time.NewTicker(config.Pending_resend_delay),
	}
	elevator.Door_timer.Stop()
	elevator.Door_timer_done = false

	return elevator
}

func (e *Elevator) Backup_pending() {
	data, _ := json.Marshal(e.Pending)
	os.WriteFile("pending_backup.json", data, 0644)
}

func (e *Elevator) Timer_handler(msgChan chan<- network.Message, lossChan chan<- *network.Client, quitChan chan<- struct{}) {
	for {
		select {
			case <-e.Reconnect_timer.C:
				e.Retry_counter++
				log.Printf("Retyr counter: %d \n", e.Retry_counter)
				e.Reconnect_timer.Stop()

				if e.Retry_counter > config.Max_retries && e.Succesor {
					close(quitChan)
					return
				}

				addr, err := network.Discover_server()
				if err != nil {
					e.Reconnect_timer.Reset(config.Reconnect_delay)
					continue
				}
				conn, err := network.Connect(addr)
				if err != nil {
					e.Reconnect_timer.Reset(config.Reconnect_delay)
					continue
				}

				e.Retry_counter = 0 
				e.Connection = network.New_client(conn)
				e.Connected = true
				e.Connection.Send(network.Message{Header: network.ClientInfo, 
														Payload: &network.DataPayload{ID: e.Id, CurrentFloor: e.Current_floor, Obstruction: e.Obstruction}})
				go e.Connection.Listen(msgChan, lossChan)
				go e.Connection.Heart_beat()

			case <-e.Pending_ticker.C:
				if e.Connected {
					for _, message := range e.Pending {
						e.Connection.Send(*message)
					}
				}
		}
	}
}

