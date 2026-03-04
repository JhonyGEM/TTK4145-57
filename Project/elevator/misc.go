package elevator

import (
	"encoding/json"
	"log"
	"os"
	"project/config"
	"project/elevio"
	"project/network"
	"sync"
	"time"
)

func (e *Elevator) UpdateLights(request [][]bool) {
	for floor := 0; floor < config.N_floors; floor++ {
		for btn := elevio.ButtonType(0); btn < config.N_buttons; btn++ {
			elevio.SetButtonLamp(btn, floor, request[floor][btn])
		}
	}
}

func (e *Elevator) Send(message network.Message) {
	if e.Connection.Conn != nil {
		e.Connection.Send(message)
	}
}

// Handles the reconnecting routine and resending of the messages that have not been acknowledged within time limit
func (e *Elevator) TimerHandler(msgChan chan<- network.Message, lossChan chan<- *network.Client, quitChan chan<- struct{}, pendChan chan<- string, wg *sync.WaitGroup) {
	for {
		select {
			case <-e.ReconnectTimer.C:
				e.RetryCounter++
				e.ReconnectTimer.Stop()

				if e.RetryCounter > config.Max_retries && e.IsSuccesor && network.HasInternetConnection() {
					log.Print("Max tries reached, elevator becomes master \n")
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
				log.Printf("Connected to server \n")

			case <-e.PendingTicker.C:
				if e.IsConnected && len(e.Pending) > 0 {
					for _, pend_msg := range e.Pending {
						if time.Since(pend_msg.Timestamp) > config.Pending_timeout {
							pendChan <- pend_msg.Message.UID
						}
					}
				}
		}
	}
}

func (e *Elevator) SavePending() {
	data, _ := json.Marshal(e.Pending)
	os.WriteFile("pending_backup.json", data, 0644)
}

func (e *Elevator) LoadPending() {
	data, err := os.ReadFile("pending_backup.json")
	if err != nil {
		return
	}

	var pending map[string]*Pending
	if err := json.Unmarshal(data, &pending); err != nil {
		return
	}

	e.Pending = pending
}