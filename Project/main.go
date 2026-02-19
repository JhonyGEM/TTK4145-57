package main

import (
	"flag"
	"log"
	"project/config"
	"project/elevator"
	"project/elevio"
	mainMessages "project/main_functions"
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

	backup := &mainMessages.Backup{
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
					FloorHandler(floor, e)

				case btn := <-drv_buttons:
					ButtonHandler(btn, prev_btn, e)

				case obs := <-drv_obstruction:
					ObstructionHandler(obs, e)

				case <-e.Door_timer.C:
					DoorTimerHandler(e)

				case <-lossChan:
					LossConnectionHandler(e)

				case message := <-msgChan:
					log.Printf("Recieved message with header: %v", message.Header)
					switch message.Header {
					case network.OrderReceived:
						mainMessages.OrderReceivedMessage(e, message)

					case network.LightUpdate:
						e.Update_lights(message.Payload.Lights)

					case network.Backup:
						mainMessages.BackupMessage(backup, e, message)

					case network.Ack:
						mainMessages.ACKHandler(e, message)

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

			mast := master.New_master()
			mast.Cab_requests = backup.BackupCabReg
			mast.Hall_requests = backup.BackupHallReg

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.N_elevators*2)

			go network.Start_server(lossChan, newChan, msgChan)
			go mast.Client_timer_handler(lossChan)

			for {
				select {
				case new := <-newChan:
					mast.Add_client(new)
					if len(mast.Client_list) == 1 {
						mast.Successor_addr = new.Addr
						mast.Client_list[new.Addr].Send(network.Message{Header: network.Successor})
					}

				case lost := <-lossChan:
					removeClient(mast, lost)

				case msg := <-msgChan:
					log.Printf("Recived message from %s with header: %v", msg.Address, msg.Header)
					switch msg.Header {
					case network.OrderReceived:
						if msg.Payload.OrderButton == elevio.BT_Cab {
							mast.Cab_requests[msg.Payload.OrderFloor][mast.Client_list[msg.Address].ID] = true
						} else {
							mast.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = true
						}
						mast.Client_list[mast.Successor_addr].Send(network.Message{Header: network.Backup, Payload: &network.DataPayload{BackupHall: mast.Hall_requests, BackupCab: mast.Cab_requests}, UID: msg.UID})
						mast.Pending[msg.UID] = &msg

					case network.OrderFulfilled:
						if msg.Payload.OrderButton == elevio.BT_Cab {
							mast.Cab_requests[msg.Payload.OrderFloor][mast.Client_list[msg.Address].ID] = false
						} else {
							mast.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = false
							mast.Hall_assignments[msg.Payload.OrderFloor][msg.Payload.OrderButton] = ""
						}
						mast.Send_light_update()

						if mast.Still_busy(msg.Address) {
							mast.Client_list[msg.Address].Busy = false
							mast.Client_list[msg.Address].Task_timer.Stop()
						} else {
							mast.Client_list[msg.Address].Task_timer.Reset(config.Request_timeout)
						}

					case network.Ack:
						message := mast.Pending[msg.UID]

						_, ok := mast.Pending[msg.UID]
						if ok {
							mast.Distribute_request(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address)
							mast.Client_list[message.Address].Send(network.Message{Header: network.Ack, UID: message.UID})
							delete(mast.Pending, msg.UID)
						}

					case network.FloorUpdate:
						mast.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor

					case network.ObstructionUpdate:
						mast.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction

					case network.ClientInfo:
						mast.Client_list[msg.Address].ID = msg.Payload.ID
						mast.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor
						mast.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction
						mast.Resend_cab_request(msg.Address)
					}

				case <-mast.Resend_ticker.C:
					if len(mast.Client_list) > 0 {
						mast.Resend_hall_request()
					}
				}
			}
		}
	}
}

func addNewClient(m *master.Master, conn *network.Client) {
	if len(m.Client_list) == 0 {
		m.Successor_addr = conn.Addr
	}
	m.Add_client(conn)
}

func removeClient(m *master.Master, conn *network.Client) {
	id := m.Client_list[conn.Addr].ID
	m.Remove_client(conn.Addr)

	if len(m.Client_list) == 0 {
		panic("Loss of connectivity")
	} else {
		if conn.Addr == m.Successor_addr {
			for _, c := range m.Client_list {
				m.Successor_addr = c.Connection.Addr
				break
			}
		}

		m.Redistribute_request(id)
	}
}

func FloorHandler(floor int, elevator *elevator.Elevator) {
	if floor != -1 {
		elevator.Current_floor = floor
		elevio.SetFloorIndicator(floor)
		elevator.Step_FSM()
		if elevator.Connected {
			elevator.Connection.Send(network.Message{
				Header: network.FloorUpdate,
				Payload: &network.DataPayload{
					CurrentFloor: elevator.Current_floor,
				},
			})
		}
	}
}

func ButtonHandler(button elevio.ButtonEvent, previousButton elevio.ButtonEvent, elevator *elevator.Elevator) {
	if previousButton != button {
		previousButton = button
		if elevator.Connected {
			message := network.Message{
				Header: network.OrderReceived,
				Payload: &network.DataPayload{
					OrderFloor:  button.Floor,
					OrderButton: button.Button,
				},
				UID: utilities.Gen_uid(elevator.Id),
			}
			elevator.Pending[message.UID] = &message
			elevator.Send(message)
		} else {
			if button.Button == elevio.BT_Cab {
				elevator.Requests[button.Floor][button.Button] = true
				elevator.Update_lights(elevator.Requests)
				elevator.Step_FSM()
			}
		}
	}
}

func ObstructionHandler(obstruction bool, elevator *elevator.Elevator) {
	elevator.Obstruction = obstruction
	elevator.Step_FSM()
	if elevator.Connected {
		elevator.Connection.Send(network.Message{
			Header: network.ObstructionUpdate,
			Payload: &network.DataPayload{
				Obstruction: elevator.Obstruction,
			},
		})
	}
}

func DoorTimerHandler(elevator *elevator.Elevator) {
	elevator.Door_timer.Stop()
	elevator.Door_timer_done = true
	elevator.Step_FSM()
}

func LossConnectionHandler(elev *elevator.Elevator) {
	elev.Connected = false
	elev.Connection = &network.Client{}
	elev.Reconnect_timer.Reset(config.Reconnect_delay)
	elev.Remove_hall_requests()
	elev.Update_state(elevator.Undefined)
	elev.Step_FSM()
}
