package elevator

import (
	"log"
	"project/config"
	"project/elevio"
	"project/network"
	"sync"
)

func (ls *LocalState) SetButtonLights() {
	for floor := 0; floor < config.N_floors; floor++ {
		for button := elevio.ButtonType(0); button < config.N_buttons; button++ {
			elevio.SetButtonLamp(button, floor, ls.Lights[floor][button] || ls.Requests[floor][button])
		}
	}
}

func (ns *NetworkState) Send(message network.Message) {
	if ns.Connection.Conn != nil {
		ns.Connection.Send(message)
	}
}

func (e *Elevator) ReconnectLoop(msgChan chan<- network.Message, lossChan chan<- *network.Client, quitChan chan<- struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-e.Timers.Reconnect.C:
			if network.HasInternetConnection() {
				e.Network.RetryCounter++
				e.Timers.Reconnect.Stop()

				if e.Network.RetryCounter > config.Max_retries && e.Network.IsSuccessor {
					log.Println("Max retries reached, elevator becomes master")
					close(quitChan)
					wg.Wait()
					elevio.Stop()
					return
				}
				log.Printf("Reconnect attempt: %d \n", e.Network.RetryCounter)

				addr, err := network.DiscoverServer()
				if err != nil {
					e.Timers.Reconnect.Reset(config.Reconnect_delay)
					continue
				}
				conn, err := network.Connect(addr)
				if err != nil {
					e.Timers.Reconnect.Reset(config.Reconnect_delay)
					continue
				}

				e.Network.RetryCounter = 0
				e.Network.Connection = network.NewClient(conn)
				e.Network.IsConnected = true
				e.Network.Send(network.Message{
					Header: network.ClientInfo,
					Payload: &network.MessagePayload{ID: e.Network.ID, CurrentFloor: e.Local.CurrentFloor, Obstruction: e.Local.Obstruction}})
				go e.Network.Connection.Listen(msgChan, lossChan)
				go e.Network.Connection.Heartbeat()
				log.Println("Connected to server")
			} else {
				log.Println("No internet connection, retrying...")
				e.Timers.Reconnect.Reset(config.Reconnect_delay)
			}
		}
	}
}
