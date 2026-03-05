package main

import (
	"flag"
	"log"
	"net"
	"project/config"
	"project/elevator"
	"project/elevio"
	"project/master"
	"project/network"
	"project/utilities"
	"sync"
	"time"
)

type MainState int

const (
	StateElevator MainState = iota
	StateMaster
)

// TODO: Problems
// System can be slow to detect client crash (heartbeat 5s delay) -> the send wrapper funcion can crash master
// If master do not have successor, then it accepts orders but do not distribute them

// TODO: Need to do
// 1. Imporve code quality

func main() {
	state := StateElevator

	id := flag.String("id", "", "ID value")
	succesor := flag.Bool("succesor", false, "Succesor enable")
	flag.Parse()
	if *id == "" {
		log.Fatal("Missing required -id flag")
	}

	b := master.NewBackup()

	for {
		switch state {
		case StateElevator:
			log.Printf("Starting elevator \n")

			e := elevator.NewElevator(*id, *succesor)
			elevio.Init("localhost:15657", config.N_floors)
			prevBtn := elevio.ButtonEvent{Floor: -1, Button: -1}
			var wg sync.WaitGroup

			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors*config.N_buttons)
			drv_floors := make(chan int)
			drv_obstruction := make(chan bool)

			lossChan := make(chan *network.Client)
			msgChan := make(chan network.Message, config.Msg_buf_size)
			quitChan := make(chan struct{})
			pendChan := make(chan string)

			wg.Add(3)
			go elevio.PollButtons(drv_buttons, quitChan, &wg)
			go elevio.PollFloorSensor(drv_floors, quitChan, &wg)
			go elevio.PollObstructionSwitch(drv_obstruction, quitChan, &wg)
			go e.TimerHandler(msgChan, lossChan, quitChan, pendChan, &wg)

			e.LoadPending()
			e.StepFSM()

			for state == StateElevator {
				select {
				case floor := <-drv_floors:
					if floor != -1 {
						e.CurrentFloor = floor
						elevio.SetFloorIndicator(floor)
						e.StepFSM()
						if e.IsConnected {
							e.Send(network.Message{Header: network.FloorUpdate,
								Payload: &network.MessagePayload{CurrentFloor: e.CurrentFloor}})
						}
					}

				case btn := <-drv_buttons:
					if prevBtn != btn {
						prevBtn = btn
						if e.IsConnected {
							message := network.Message{Header: network.OrderReceived,
								Payload: &network.MessagePayload{OrderFloor: btn.Floor, OrderButton: btn.Button},
								UID:     utilities.GenUID(e.ID, e.Sequence)}
							e.Send(message)
							e.Sequence++
							e.Pending[message.UID] = &elevator.Pending{Message: message, Timestamp: time.Now()}
							e.SavePending()
						} else {
							if btn.Button == elevio.BT_Cab {
								e.Requests[btn.Floor][btn.Button] = true
								e.UpdateLights(e.Requests)
								e.StepFSM()
							}
						}
					}

				case obs := <-drv_obstruction:
					e.Obstruction = obs
					e.StepFSM()
					if e.IsConnected {
						e.Send(network.Message{Header: network.ObstructionUpdate,
							Payload: &network.MessagePayload{Obstruction: e.Obstruction}})
					}

				case <-e.DoorTimer.C:
					e.DoorTimer.Stop()
					e.DoorTimerDone = true
					e.StepFSM()

				case <-lossChan:
					log.Print("Disconnected from server \n")
					e.IsConnected = false
					e.Connection = &network.Client{}
					e.ReconnectTimer.Reset(config.Reconnect_delay)
					e.RemoveHallRequests()
					e.StepFSM()

				case message := <-msgChan:
					e.HandleMessage(message, b)

				case <-quitChan:
					state = StateMaster
					utilities.StartNewInstance(e.ID)
					continue

				case uid := <-pendChan:
					e.Send(e.Pending[uid].Message)
					e.Pending[uid].Timestamp = time.Now()
					e.SavePending()
				}
			}

		case StateMaster:
			log.Printf("Starting master \n")

			m := master.NewMaster()
			m.CabRequests = b.CabRequests
			m.HallRequests = b.HallRequests

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.Msg_buf_size)

			go network.StartServer(lossChan, newChan, msgChan)
			go m.ClientTimerHandler()

			for m.IP == "" {
				if addr, err := network.DiscoverServer(); err == nil {
					m.IP, _, _ = net.SplitHostPort(addr)
				}
			}

			for {
				select {
				case new := <-newChan:
					m.AddClient(new)
					if !m.HasSuccessor && m.IP == new.GetIP() {
						m.HasSuccessor = true
						m.SuccessorAddr = new.Addr
						m.ClientList[new.Addr].Send(network.Message{Header: network.Succesor})
					}
					m.ResendCabRequest(new.Addr)
					m.SendLightUpdate()

				case lost := <-lossChan:
					id := m.ClientList[lost.Addr].ID
					m.RemoveClient(lost.Addr)
					if len(m.ClientList) == 0 {
						log.Fatal("Loss of internet")
					} else {
						if lost.Addr == m.SuccessorAddr {
							m.HasSuccessor = false
							m.SuccessorAddr = ""
							m.FindNewSuccessor()
						}
						m.RedistributeHallRequest(id)
					}

				case message := <-msgChan:
					m.HandleMessage(message)

				case <-m.ResendTicker.C:
					if m.HasSuccessor && len(m.ClientList) > 0 {
						m.ResendHallRequest()
					}
				}
			}
		}
	}
}
