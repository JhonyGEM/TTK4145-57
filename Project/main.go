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
// If master do not have successor, then it accepts orders but does not distribute them or update the button lights
// Can we end up in situation where there is no successor?

// TODO: Recent fixes
// fusion of offline and master cab lights, init with obstruction active, init with active orders, some init elev logic, added redundancy to successor mesgs, saving and loading of cab request durning elev promotion

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
					e.HandleFloorUpdate(floor)

				case btn := <-drv_buttons:
					if prevBtn != btn {
						prevBtn = btn
						e.HanldleButtonPress(btn)
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
					e.HandleDisconnect()

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
					if !m.HasSuccessor && !m.SuccessorNotified {
						m.NotifyNewSuccessor()
					}
					m.ResendCabRequest(new.Addr)
					m.SendLightUpdate()

				case lost := <-lossChan:
					m.HandleClientLoss(lost)

				case message := <-msgChan:
					m.HandleMessage(message)

				case <-m.ResendTicker.C:
					if m.HasSuccessor && len(m.ClientList) > 0 {
						m.ResendHallRequest()
					}

				case <-m.SuccessorTimeout.C:
					m.SuccessorTimeout.Stop()
					if !m.HasSuccessor {
						m.IsSuccessorTimeout = true
						m.NotifyNewSuccessor()
					}
				}
			}
		}
	}
}
