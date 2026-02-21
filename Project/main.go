package main

import (
	"flag"
	"log"
	"project/config"
	"project/elevator"
	"project/elevio"
	"project/master"
	"project/network"
	"project/utilities"
	"sync"
)

type MainState int

const (
	StateElevator MainState = iota
	StateMaster
)

// TODO: Problems


// TODO: Need to do
// 1. Imporve code quality
// 2. Test if everyting thats implemented works
// 3. Improve find_closest_elevator

func main() {
	state := StateElevator

	succesor := flag.Bool("succesor", false, "Succesor enable")
	id := flag.String("id", "", "ID value")
	flag.Parse()

	b := master.New_backup()

	for {
		switch  state {
		case StateElevator:
			log.Printf("Starting elevator \n")

			e := elevator.New_elevator(*id)
			elevio.Init("localhost:15657", config.N_floors)
			e.Succesor = *succesor
			prev_btn := elevio.ButtonEvent{Floor: -1, Button: -1}
			var wg sync.WaitGroup

			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors * config.N_buttons)
			drv_floors := make(chan int)
			drv_obstruction := make(chan bool)

			lossChan := make(chan *network.Client)
			msgChan := make(chan network.Message, config.Msg_buf_size)
			quitChan := make(chan struct{})

			wg.Add(3)
			go elevio.PollButtons(drv_buttons, quitChan, &wg)
			go elevio.PollFloorSensor(drv_floors, quitChan, &wg)
			go elevio.PollObstructionSwitch(drv_obstruction, quitChan, &wg)
			go e.Timer_handler(msgChan, lossChan, quitChan, &wg)

			e.Load_pending()
			e.Step_FSM()

			for state == StateElevator {
				select {
				case floor := <-drv_floors:
					if floor != -1 {
						e.Current_floor = floor
						elevio.SetFloorIndicator(floor)
						e.Step_FSM()
						if e.Connected {
							e.Send(network.Message{Header: network.FloorUpdate, 
												   Payload: &network.MessagePayload{CurrentFloor: e.Current_floor}})
						}
					}
				
				case btn := <-drv_buttons:
					if prev_btn != btn {
						prev_btn = btn
						if e.Connected {
							message := network.Message{Header: network.OrderReceived, Payload: &network.MessagePayload{OrderFloor: btn.Floor, OrderButton: btn.Button}, UID: utilities.Gen_uid(e.Id, e.Sequence)}
							e.Sequence++
							e.Pending[message.UID] = &message
							e.Send(message)
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
						e.Send(network.Message{Header: network.ObstructionUpdate, 
											   Payload: &network.MessagePayload{Obstruction: e.Obstruction}})
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
					e.Handle_message(message, b)

				case <-quitChan:
					state = StateMaster
					utilities.Start_new_instance(e.Id)
					continue
				}
			}

		case StateMaster:
			log.Printf("Starting master \n")

			m := master.New_master()
			m.Cab_requests = b.Cab_req
			m.Hall_requests = b.Hall_req

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.Msg_buf_size)
			

			go network.Start_server(lossChan, newChan, msgChan)
			go m.Client_timer_handler(lossChan)

			for {
				select {
					case new := <-newChan:
						m.Add_client(new)
						if len(m.Client_list) == 1 {
							m.Successor_addr = new.Addr
							m.Client_list[new.Addr].Send(network.Message{Header: network.Succesor})
						}
						m.Resend_cab_request(new.Addr)
						m.Send_light_update()

					case lost := <-lossChan:
						id := m.Client_list[lost.Addr].ID
						m.Remove_client(lost.Addr)
						if len(m.Client_list) == 0 {
							panic("Loss of internet")
						} else {
							if lost.Addr == m.Successor_addr {
								for _, c := range m.Client_list {
									m.Successor_addr = c.Connection.Addr
									c.Send(network.Message{Header: network.Succesor})
									break
								}
							}
							m.Redistribute_hall_request(id)
						}

					case message := <-msgChan:
						m.Handle_message(message)

					case <-m.Resend_ticker.C:
						if len(m.Client_list) > 0 {
							m.Resend_hall_request() 
						} 
				}
			}
		}
	}
}