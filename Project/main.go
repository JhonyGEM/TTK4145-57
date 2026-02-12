package main

import (
	"config"
	"elevator"
	"elevio"
	"log"
	"master"
	"network"
	"utilities"
)

type PairState int

const (
	StateElevator PairState = iota
	StateMaster
)

func main() {
	state := StateElevator

	backup_hall_reg := utilities.Create_request_arr(config.N_floors, config.N_buttons)
	backup_cab_reg := master.Create_cab_requests(config.N_floors)

	for {
		switch state {
		case StateElevator:
			log.Printf("Starting elevator \n")

			// "Pop " is the id
			e := elevator.New_elevator("pop")
			// Change this hardcoded address
			elevio.Init("localhost:15657", config.N_floors)

			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors*config.N_buttons)
			drv_floors := make(chan int)
			drv_obstruction := make(chan bool)

			lossChan := make(chan *network.Client)
			msgChan := make(chan network.Message)
			quitChan := make(chan struct{})
			var prev_btn elevio.ButtonEvent

			go elevio.PollButtons(drv_buttons, quitChan)
			go elevio.PollFloorSensor(drv_floors, quitChan)
			go elevio.PollObstructionSwitch(drv_obstruction, quitChan)

			go e.Timer_handler(msgChan, lossChan, quitChan)
			e.Load_pending()
			log.Printf("Length: %d", len(e.Pending))
			e.Step_FSM()

			for state == StateElevator {
				select {
				case floor := <-drv_floors:
					if floor != -1 {
						e.Current_floor = floor
						elevio.SetFloorIndicator(floor)
						e.Step_FSM()
						if e.Connected {
							e.Connection.Send(network.Message{
								Header: network.FloorUpdate,
								Payload: &network.DataPayload{
									CurrentFloor: e.Current_floor,
								},
							})
						}
					}

				case btn := <-drv_buttons:
					if prev_btn != btn {
						prev_btn = btn
						if e.Connected {
							message := network.Message{
								Header: network.OrderReceived,
								Payload: &network.DataPayload{
									OrderFloor:  btn.Floor,
									OrderButton: btn.Button,
								},
								UID: utilities.Gen_uid(e.Id),
							}
							e.Pending[message.UID] = &message
							e.Connection.Send(message)
						} else {
							if btn.Button == elevio.BT_Cab {
								e.Requests[btn.Floor][btn.Button] = true
								e.Update_lights(e.Requests)
								e.Step_FSM()
							}
						}
					}

				case obs := <-drv_obstruction:
					e.Obstruction = obs
					e.Step_FSM()
					if e.Connected {
						e.Connection.Send(network.Message{
							Header: network.ObstructionUpdate,
							Payload: &network.DataPayload{
								Obstruction: e.Obstruction,
							},
						})
					}

				case <-e.Door_timer.C:
					e.Door_timer.Stop()
					e.Door_timer_done = true
					e.Step_FSM()

				case <-lossChan:
					e.Connected = false
					e.Connection = &network.Client{}
					e.Reconnect_timer.Reset(config.Reconnect_delay)
					e.Remove_hall_requests()
					e.Update_state(elevator.Undefined)
					e.Step_FSM()

				case message := <-msgChan:
					log.Printf("Recieved message")
					switch message.Header {
					case network.OrderReceived:
						e.Requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
						e.Save_pending()
						e.Update_lights(e.Requests)
						e.Step_FSM()

					case network.LightUpdate:
						e.Update_lights(message.Payload.Lights)

					case network.Backup:
						log.Printf("Recieved backup")
						backup_cab_reg = message.Payload.BackupCab
						backup_hall_reg = message.Payload.BackupHall
						e.Connection.Send(network.Message{
							Header: network.Ack,
							UID:    message.UID,
						})

					case network.Ack:
						_, ok := e.Pending[message.UID]
						if ok {
							delete(e.Pending, message.UID)
							e.Save_pending()
						}

					}
				case <-quitChan:
					state = StateMaster
					utilities.Start_new_instance()
					continue
				}
			}

		case StateMaster:
			log.Printf("Starting master \n")

			master := master.New_master()
			master.Cab_requests = backup_cab_reg
			master.Hall_requests = backup_hall_reg

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.N_elevators*2)

			go network.Start_server(lossChan, newChan, msgChan)
			go master.Client_timer_handler(lossChan)

			for {
				select {
				case new := <-newChan:
					addNewClient(master, new)

				case lost := <-lossChan:
					removeClient(master, lost)

				case msg := <-msgChan:
					log.Printf("Recived message from %s with header: %d", msg.Address, msg.Header)
					switch msg.Header {
					case network.OrderReceived:
						if msg.Payload.OrderButton == elevio.BT_Cab {
							master.Cab_requests[msg.Payload.OrderFloor][master.Client_list[msg.Address].ID] = true
						} else {
							master.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = true
						}
						master.Client_list[master.Successor_addr].Connection.Send(network.Message{Header: network.Backup, Payload: &network.DataPayload{BackupHall: master.Hall_requests, BackupCab: master.Cab_requests}, UID: msg.UID})
						master.Pending[msg.UID] = &msg

					case network.OrderFulfilled:
						if msg.Payload.OrderButton == elevio.BT_Cab {
							master.Cab_requests[msg.Payload.OrderFloor][master.Client_list[msg.Address].ID] = false
						} else {
							master.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = false
							master.Hall_assignments[msg.Payload.OrderFloor][msg.Payload.OrderButton] = ""
						}

						if master.Should_clear_busy(msg.Address) {
							master.Client_list[msg.Address].Busy = false
							master.Client_list[msg.Address].Task_timer.Stop()
						} else {
							master.Client_list[msg.Address].Task_timer.Reset(config.Request_timeout)
						}

					case network.Ack:
						message := master.Pending[msg.UID]

						_, ok := master.Pending[msg.UID]
						if ok {
							master.Distribute_request(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address)
							master.Client_list[message.Address].Connection.Send(network.Message{Header: network.Ack, UID: message.UID})
							delete(master.Pending, msg.UID)
						}

					case network.FloorUpdate:
						master.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor

					case network.ObstructionUpdate:
						master.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction

					case network.ClientInfo:
						master.Client_list[msg.Address].ID = msg.Payload.ID
						master.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor
						master.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction
					}
				}
			}
		}
	}
}

func addNewClient(m *master.Master, conn *network.Client) {
	if len(master.Client_list) == 0 {
		master.Successor_addr = new.Addr
	}
	master.Add_client(new)
}

func removeClient(m *master.Master, conn *network.Client) {
	id := master.Client_list[conn.Addr].ID
	master.Remove_client(conn.Addr)

	if len(master.Client_list) == 0 {
		panic("Loss of connectivity")
	} else {
		if conn.Addr == master.Successor_addr {
			for _, c := range master.Client_list {
				master.Successor_addr = c.Connection.Addr
				break
			}
		}

		master.Redistribute_request(id)
	}
}
