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

			buttonChan := make(chan elevio.ButtonEvent, config.Btn_buf_size)
			floorChan := make(chan int)
			obstructionChan := make(chan bool)

			lostChan := make(chan *network.Client)
			messageChan := make(chan network.Message, config.Msg_buf_size)
			quitChan := make(chan struct{})

			wg.Add(3)
			go elevio.PollButtons(buttonChan, quitChan, &wg)
			go elevio.PollFloorSensor(floorChan, quitChan, &wg)
			go elevio.PollObstructionSwitch(obstructionChan, quitChan, &wg)
			go e.ReconnectLoop(messageChan, lostChan, quitChan, &wg)

			e.Network.Pending = utilities.LoadFromFile(config.Pending_backup)
			e.Local.LoadCabRequests()
			e.StepFSM()

			for role == RoleElevator {
				select {
				case floor := <-floorChan:
					e.HandleFloorUpdate(floor)

				case button := <-buttonChan:
					if previousButton != button {
						previousButton = button
						e.HandleButtonPress(button)
					}

				case obstruction := <-obstructionChan:
					e.HandleObstructionUpdate(obstruction)

				case <-e.Timers.Door.C:
					e.Timers.Door.Stop()
					e.Local.DoorTimerDone = true
					e.StepFSM()

				case <-lostChan:
					e.HandleDisconnect()

				case message := <-messageChan:
					e.HandleMessage(message, b)

				case <-quitChan:
					e.SaveCabRequests()
					role = RoleMaster
					utilities.StartNewInstance(e.Network.ID)
					continue

				case <-e.Timers.PendingTicker.C:
					e.Network.ResendPendingRequest()
				}
			}

		case RoleMaster:
			log.Println("Starting master")

			m := master.NewMaster()
			m.Requests.Cab = b.CabRequests
			m.Requests.Hall = b.HallRequests
			m.IP = network.GetLocalIP()

			newClientChan := make(chan *network.Client, config.N_elevators)
			lostClientChan := make(chan *network.Client, config.N_elevators)
			messageChan := make(chan network.Message, config.Msg_buf_size)

			go network.StartServer(lostClientChan, newClientChan, messageChan)

			for {
				select {
				case new := <-newClientChan:
					m.HandleNewClient(new)

				case lost := <-lostClientChan:
					m.HandleClientLoss(lost)

				case message := <-messageChan:
					m.HandleMessage(message)

				case <-m.Tickers.Resend.C:
					if m.HasSuccessor {
						m.ResendHallRequest()
					}

				case <-m.Tickers.Timeout.C:
					m.HandleClientTimeout()
					m.HandleSuccessorAckTimeout()

				case <-m.Successor.SearchTimer.C:
					m.Successor.SearchTimer.Stop()
					if !m.HasSuccessor {
						m.Successor.SearchTimedOut = true
						m.NotifyNewSuccessor()
					}
				}
			}
		}
	}
}
