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

type Role int

const (
	RoleElevator Role = iota
	RoleMaster
)

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
			log.Printf("Starting elevator with id: %s", *id)

			e := elevator.NewElevator(*id, *successor)
			elevio.Init("localhost:15657", config.N_floors)
			previousButton := elevio.ButtonEvent{Floor: -1, Button: -1}
			var wg sync.WaitGroup

			drv_buttons := make(chan elevio.ButtonEvent, config.N_floors*config.N_buttons)
			drv_floors := make(chan int)
			drv_obstruction := make(chan bool)

			lostChan := make(chan *network.Client)
			msgChan := make(chan network.Message, config.Msg_buf_size)
			quitChan := make(chan struct{})

			wg.Add(3)
			go elevio.PollButtons(drv_buttons, quitChan, &wg)
			go elevio.PollFloorSensor(drv_floors, quitChan, &wg)
			go elevio.PollObstructionSwitch(drv_obstruction, quitChan, &wg)
			go e.ReconnectLoop(msgChan, lostChan, quitChan, &wg)

			e.Pending = utilities.LoadFromFile(config.Pending_backup)
			e.LoadCabRequests()
			e.StepFSM()

			for role == RoleElevator {
				select {
				case floor := <-drv_floors:
					e.HandleFloorUpdate(floor)

				case button := <-drv_buttons:
					if previousButton != button {
						previousButton = button
						e.HandleButtonPress(button)
					}

				case obstruction := <-drv_obstruction:
					e.HandleObstructionUpdate(obstruction)

				case <-e.DoorTimer.C:
					e.DoorTimer.Stop()
					e.DoorTimerDone = true
					e.StepFSM()

				case <-lostChan:
					e.HandleDisconnect()

				case message := <-msgChan:
					e.HandleMessage(message, b)

				case <-quitChan:
					e.SaveCabRequests()
					role = RoleMaster
					utilities.StartNewInstance(e.ID)
					continue

				case <-e.PendingTicker.C:
					e.ResendPendingRequest()
				}
			}

		case RoleMaster:
			log.Println("Starting master")

			m := master.NewMaster()
			m.CabRequests = b.CabRequests
			m.HallRequests = b.HallRequests
			m.IP = network.GetLocalIP()

			newChan := make(chan *network.Client, config.N_elevators)
			lostChan := make(chan *network.Client, config.N_elevators)
			msgChan := make(chan network.Message, config.Msg_buf_size)

			go network.StartServer(lostChan, newChan, msgChan)

			for {
				select {
				case new := <-newChan:
					m.HandleNewClient(new)

				case lost := <-lostChan:
					m.HandleClientLoss(lost)

				case message := <-msgChan:
					m.HandleMessage(message)

				case <-m.ResendTicker.C:
					if m.HasSuccessor {
						m.ResendHallRequest()
					}

				case <-m.TimeoutTicker.C:
					m.HandleClientTimeout()
					m.HandleSuccessorAckTimeout()

				case <-m.Successor.TimeoutTimer.C:
					m.Successor.TimeoutTimer.Stop()
					if !m.HasSuccessor {
						m.Successor.IsTimeout = true
						m.NotifyNewSuccessor()
					}
				}
			}
		}
	}
}
