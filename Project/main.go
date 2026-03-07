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
	"time"
)

type Role int

const (
	RoleElevator Role = iota
	RoleMaster
)

// TODO: Problems
// (not a problem) System can be slow to detect client crash (heartbeat 5s delay) -> the send wrapper funcion can crash master
// (not a problem?) If master do not have successor, then it accepts orders but do not distribute them
// Can we end up in situation where there is no successor?
// fixed fusion of offline and master cab lights, init with obstruction active, init with active orders, some init elev logic, added redundancy to successor mesgs, saving and loading of cab request durning elev promotion

// TODO: Need to do
// 1. Imporve code quality

func main() {
	role := RoleElevator

	id := flag.String("id", "", "ID value")
	successor := flag.Bool("successor", false, "Successor enable")
	flag.Parse()
	if *id == "" {
		log.Fatal("Missing required -id flag")
	}

	b := master.NewBackup()

	for {
		switch role {
		case RoleElevator:
			log.Println("Starting elevator")

			e := elevator.NewElevator(*id, *successor)
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

			e.Pending = utilities.LoadFromFile(config.Pending_backup)
			e.LoadCabRequests()
			e.StepFSM()

			for role == RoleElevator {
				select {
				case floor := <-drv_floors:
					if floor != -1 {
						e.CurrentFloor = floor
						elevio.SetFloorIndicator(floor)
						e.StepFSM()
						if e.IsConnected {
							e.Send(network.Message{
								Header: network.FloorUpdate,
								Payload: &network.MessagePayload{
									CurrentFloor: e.CurrentFloor}})
						}
					}

				case btn := <-drv_buttons:
					if prevBtn != btn {
						prevBtn = btn
						if e.IsConnected {
							message := network.Message{
								Header: network.OrderReceived,
								Payload: &network.MessagePayload{
									OrderFloor: btn.Floor, OrderButton: btn.Button},
								UID: utilities.GenUID(e.ID, e.Sequence)}
							e.Sequence++				
							e.Pending[message.UID] = &network.Pending{Message: message, Timestamp: time.Now()}
							utilities.SaveToFile(config.Pending_backup, e.Pending)
							e.Send(message)
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
						e.Send(network.Message{
							Header: network.ObstructionUpdate,
							Payload: &network.MessagePayload{
								Obstruction: e.Obstruction}})
					}

				case <-e.DoorTimer.C:
					e.DoorTimer.Stop()
					e.DoorTimerDone = true
					e.StepFSM()

				case <-lossChan:
					log.Println("Disconnected from server")
					e.IsConnected = false
					e.Connection = &network.Client{}
					e.ReconnectTimer.Reset(config.Reconnect_delay)
					e.RemoveHallRequests()
					e.UpdateLights(e.Requests)
					if elevio.GetFloor() == -1 && !e.RequestPending() {
						e.UpdateState(elevator.Undefined)
					}
					e.StepFSM()

				case message := <-msgChan:
					e.HandleMessage(message, b)

				case <-quitChan:
					e.SaveCabRequests()
					role = RoleMaster
					utilities.StartNewInstance(e.ID)
					continue

				case uid := <-pendChan:
					e.Send(e.Pending[uid].Message)
					e.Pending[uid].Timestamp = time.Now()
					utilities.SaveToFile(config.Pending_backup, e.Pending)
				}
			}

		case RoleMaster:
			log.Println("Starting master")

			m := master.NewMaster()
			m.CabRequests = b.CabRequests
			m.HallRequests = b.HallRequests
			m.IP = network.GetLocalIP()

			newChan := make(chan *network.Client, config.N_elevators)
			lossChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.Msg_buf_size)

			go network.StartServer(lossChan, newChan, msgChan)
			go m.MonitorTimeouts()

			for {
				select {
				case new := <-newChan:
					m.AddClient(new)
					if !m.HasSuccessor && !m.ProspectNotified {
						m.FindNewSuccessor()
					}
					m.ResendCabRequest(new.Addr)
					m.SendLightUpdate()

				case lost := <-lossChan:
					id := m.ClientList[lost.Addr].ID
					m.RemoveClient(lost.Addr)
					if len(m.ClientList) == 0 {
						log.Fatal("Loss of internet")
					} 
					if len(m.ClientList) == 1 && !network.HasInternetConnection() {
						for _, client := range m.ClientList {
							client.Connection.Conn.Close()
							break
						}
					}
					if len(m.ClientList) > 0 {
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
