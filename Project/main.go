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
		switch  state {
		case StateElevator:
			log.Printf("Starting elevator \n")

			e := elevator.New_elevator("pop")
			elevio.Init("localhost:15657", config.N_floors)
			
			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors * config.N_buttons)
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
							e.Connection.Send(network.Message{Header: network.FloorUpdate, 
																	Payload: &network.DataPayload{CurrentFloor: e.Current_floor}})
						}
					}
				
				case btn := <-drv_buttons:
					if prev_btn != btn {
						prev_btn = btn
						if e.Connected {
							message := network.Message{Header: network.OrderReceived, Payload: &network.DataPayload{OrderFloor: btn.Floor, OrderButton: btn.Button}, UID: utilities.Gen_uid(e.Id)}
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
						e.Connection.Send(network.Message{Header: network.ObstructionUpdate, 
																Payload: &network.DataPayload{Obstruction: e.Obstruction}})
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
						e.Connection.Send(network.Message{Header: network.Ack, UID: message.UID})

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

			m := master.New_master()
			m.Cab_requests = backup_cab_reg
			m.Hall_requests = backup_hall_reg

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.N_elevators * 2)
			

			go network.Start_server(lossChan, newChan, msgChan)
			go m.Client_timer_handler(lossChan)

			for {
				select {
					case new := <-newChan:
						if len(m.Client_list) == 0 {
							m.Successor_addr = new.Addr
						}
						m.Add_client(new)

					case lost := <-lossChan:
						id := m.Client_list[lost.Addr].ID
						m.Remove_client(lost.Addr)
						if len(m.Client_list) == 0 {
							panic("Loss of internet")
						} else {
							if lost.Addr == m.Successor_addr {
								for _, c := range m.Client_list {
									m.Successor_addr = c.Connection.Addr
									break
								}
							}
							m.Redistribute_request(id)
						}

					case msg := <-msgChan:
						log.Printf("Recived message from %s with header: %d", msg.Address, msg.Header)
						switch msg.Header {
						case network.OrderReceived:
							if msg.Payload.OrderButton == elevio.BT_Cab {
								m.Cab_requests[msg.Payload.OrderFloor][m.Client_list[msg.Address].ID] = true
							} else {
								m.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = true
							}
							m.Client_list[m.Successor_addr].Connection.Send(network.Message{Header: network.Backup, Payload: &network.DataPayload{BackupHall: m.Hall_requests, BackupCab: m.Cab_requests}, UID: msg.UID})
							m.Pending[msg.UID] = &msg

						case network.OrderFulfilled:
							if msg.Payload.OrderButton == elevio.BT_Cab {
								m.Cab_requests[msg.Payload.OrderFloor][m.Client_list[msg.Address].ID] = false
							} else {
								m.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = false
								m.Hall_assignments[msg.Payload.OrderFloor][msg.Payload.OrderButton] = ""
								m.Send_light_update()
							}

							if m.Still_busy(msg.Address) {
								m.Client_list[msg.Address].Busy = false
								m.Client_list[msg.Address].Task_timer.Stop()
							} else {
								m.Client_list[msg.Address].Task_timer.Reset(config.Request_timeout)
							}

						case network.Ack:
							message := m.Pending[msg.UID]

							_, ok := m.Pending[msg.UID]
							if ok {
								m.Distribute_request(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address)
								m.Client_list[message.Address].Connection.Send(network.Message{Header: network.Ack, UID: message.UID})
								delete(m.Pending, msg.UID)
							}

						case network.FloorUpdate:
							m.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor

						case network.ObstructionUpdate:
							m.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction

						case network.ClientInfo:
							m.Client_list[msg.Address].ID = msg.Payload.ID
							m.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor
							m.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction
						}
				}
			}
		}
	}
}