package main

import (
	"flag"
	"log"
	"project/config"
	"project/elevator"
	"project/elevio"
	"project/mainfunctions"
	"project/master"
	"project/network"
	"project/utilities"
)

type PairState int

const (
	StateElevator PairState = iota
	StateMaster
)

// TODO: Problems

// TODO: Need to do
// 1. Imporve code quality
// 2. Test if everyting thats implemented works

func main() {
	state := StateElevator

	Successor := flag.Bool("Successor", false, "Successor enable")
	id := flag.String("id", "", "ID value")
	flag.Parse()

	backup := &mainfunctions.Backup{
		BackupHallReg: utilities.Create_request_arr(config.N_floors, config.N_buttons),
		BackupCabReg:  master.Create_cab_requests(config.N_floors),
	}

	for {
		switch state {
		case StateElevator:
			log.Printf("Starting elevator \n")

			e := elevator.New_elevator(*id)
			elevio.Init("localhost:15657", config.N_floors)
			e.Successor = *Successor
			prev_btn := elevio.ButtonEvent{Floor: -1, Button: -1}

			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors*config.N_buttons)
			drv_floors := make(chan int)
			drv_obstruction := make(chan bool)

			lossChan := make(chan *network.Client)
			msgChan := make(chan network.Message)
			quitChan := make(chan struct{})

			go elevio.PollButtons(drv_buttons, quitChan)
			go elevio.PollFloorSensor(drv_floors, quitChan)
			go elevio.PollObstructionSwitch(drv_obstruction, quitChan)
			go e.Timer_handler(msgChan, lossChan, quitChan)

			e.Load_pending()
			e.Step_FSM()

			for state == StateElevator {
				select {
				case floor := <-drv_floors:
					mainfunctions.FloorHandler(floor, e)

				case btn := <-drv_buttons:
					mainfunctions.ButtonHandler(btn, prev_btn, e)

				case obs := <-drv_obstruction:
					mainfunctions.ObstructionHandler(obs, e)

				case <-e.Door_timer.C:
					mainfunctions.DoorTimerHandler(e)

				case <-lossChan:
					mainfunctions.LossConnectionHandler(e)

				case message := <-msgChan:
					log.Printf("Recieved message with header: %v", message.Header)
					switch message.Header {
					case network.OrderReceived:
						mainfunctions.ElevatorOrderReceivedMessage(e, message)

					case network.LightUpdate:
						e.Update_lights(message.Payload.Lights)

					case network.Backup:
						mainfunctions.BackupMessage(backup, e, message)

					case network.Ack:
						mainfunctions.ElevatorACKHandler(e, message)

					case network.Successor:
						e.Successor = true
					}

				case <-quitChan:
					state = StateMaster
					utilities.Start_new_instance(e.Id)
					continue
				}
			}

		case StateMaster:
			log.Printf("Starting master \n")

			m := master.New_master()
			m.Cab_requests = backup.BackupCabReg
			m.Hall_requests = backup.BackupHallReg

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.N_elevators*2)

			go network.Start_server(lossChan, newChan, msgChan)
			go m.Client_timer_handler(lossChan)

			for {
				select {
				case new := <-newChan:
					mainfunctions.AddElevatorHandler(m, new)

				case lost := <-lossChan:
					mainfunctions.RemoveClient(m, lost)

				case msg := <-msgChan:
					log.Printf("Recived message from %s with header: %v", msg.Address, msg.Header)
					switch msg.Header {
					case network.OrderReceived:
						mainfunctions.MasterOrderReceivedMessage(m, msg)

					case network.OrderFulfilled:
						mainfunctions.OrderFulfilledMessage(m, msg)

					case network.Ack:
						mainfunctions.MasterACKHandler(m, msg)

					case network.FloorUpdate:
						m.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor

					case network.ObstructionUpdate:
						m.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction

					case network.ClientInfo:
						mainfunctions.ClientInfoMessage(m, msg)
					}

				case <-m.Resend_ticker.C:
					mainfunctions.Resend(m)
				}
			}
		}
	}
}
