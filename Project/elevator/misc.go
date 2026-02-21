package elevator

import (
	"project/config"
	"project/network"
	"project/elevio"
	"encoding/json"
	"log"
	"os"
	"sync"
)

func (e *Elevator) Update_lights(request [][]bool) {
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

// Handles the reconnecting rutin and resend of the pending array
func (e *Elevator) Timer_handler(msgChan chan<- network.Message, lossChan chan<- *network.Client, quitChan chan<- struct{}, wg *sync.WaitGroup) {
	for {
		select {
			case <-e.Reconnect_timer.C:
				e.Retry_counter++
				log.Printf("Reconnect attempt: %d \n", e.Retry_counter)
				e.Reconnect_timer.Stop()

				if e.Retry_counter > config.Max_retries && e.Succesor {
					close(quitChan)
					wg.Wait()
					elevio.Stop()
					return
				}

				addr, err := network.Discover_server()
				if err != nil {
					e.Reconnect_timer.Reset(config.Reconnect_delay)
					continue
				}
				conn, err := network.Connect(addr)
				if err != nil {
					e.Reconnect_timer.Reset(config.Reconnect_delay)
					continue
				}

				e.Retry_counter = 0 
				e.Connection = network.New_client(conn)
				e.Connected = true
				e.Send(network.Message{Header: network.ClientInfo, 
									   Payload: &network.MessagePayload{ID: e.Id, CurrentFloor: e.Current_floor, Obstruction: e.Obstruction}})
				go e.Connection.Listen(msgChan, lossChan)
				go e.Connection.Heartbeat()

			case <-e.Pending_ticker.C:
				if e.Connected {
					if len(e.Pending) > 0 {
						for _, message := range e.Pending {
							e.Send(*message)
						}
					}
				}
		}
	}
}

func (e *Elevator) Save_pending() {
	data, _ := json.Marshal(e.Pending)
	os.WriteFile("pending_backup.json", data, 0644)
}

func (e *Elevator) Load_pending() {
	data, err := os.ReadFile("pending_backup.json")
	if err != nil {
		return
	}

	var pending map[string]*network.Message
	if err := json.Unmarshal(data, &pending); err != nil {
		return
	}

	e.Pending = pending
}