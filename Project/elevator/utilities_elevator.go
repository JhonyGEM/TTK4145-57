package elevator

import (
	"log"
	"project/config"
	"project/elevio"
	"project/network"
	"sync"
)

func (e *Elevator) UpdateLights(request [][]bool) {
	for floor := 0; floor < config.N_floors; floor++ {
		for btn := elevio.ButtonType(0); btn < config.N_buttons; btn++ {
			elevio.SetButtonLamp(btn, floor, request[floor][btn] || e.Requests[floor][btn])
		}
	}
}

func (e *Elevator) Send(message network.Message) {
	if e.Connection.Conn != nil {
		e.Connection.Send(message)
	}
}

func (e *Elevator) ReconnectLoop(msgChan chan<- network.Message, lossChan chan<- *network.Client, quitChan chan<- struct{}, wg *sync.WaitGroup) {
	for {
		select {
		case <-e.ReconnectTimer.C:
			if network.HasInternetConnection() {
				e.RetryCounter++
				e.ReconnectTimer.Stop()

				if e.RetryCounter > config.Max_retries && e.IsSuccessor {
					log.Println("Max retries reached, elevator becomes master")
					close(quitChan)
					wg.Wait()
					elevio.Stop()
					return
				}
				log.Printf("Reconnect attempt: %d \n", e.RetryCounter)

				addr, err := network.DiscoverServer()
				if err != nil {
					e.ReconnectTimer.Reset(config.Reconnect_delay)
					continue
				}
				conn, err := network.Connect(addr)
				if err != nil {
					e.ReconnectTimer.Reset(config.Reconnect_delay)
					continue
				}

				e.RetryCounter = 0
				e.Connection = network.NewClient(conn)
				e.IsConnected = true
				e.Send(network.Message{Header: network.ClientInfo,
					Payload: &network.MessagePayload{ID: e.ID, CurrentFloor: e.CurrentFloor, Obstruction: e.Obstruction}})
				go e.Connection.Listen(msgChan, lossChan)
				go e.Connection.Heartbeat()
				log.Println("Connected to server")
			} else {
				log.Println("No internet connection, retrying...")
				e.ReconnectTimer.Reset(config.Reconnect_delay)
			}
		}
	}
}
