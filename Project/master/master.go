package master

import (
	"config"
	"elevio"
	"log"
	"network"
	"time"
	"utilities"
)

type Master struct {
	client_list 	     map[string]*Elevator_client // key is addr
	hall_requests        [][]bool
	hall_assignments     [][]string
	cab_requests         []map[string]bool
	successor_addr       string
}

type Elevator_client struct {
	Connection 		*network.Client
	ID 				string
	current_floor 	int
	obstruction 	bool
	busy 			bool
	task_timer 		*time.Timer
}

// TODO: Problems
// 1. elevator reconnect logic can block/delay fsm step

// TODO: Need to implement
// 1. Better file division and variable/function names
// 2. Redundancy in communication / sync of hall and cab request from master (maybe WAL or loging of messages?)
// 3. Adapt master so it can be started with synced requests
// 4. Resend order after new master takeover
// 5. Elevator need to send out OrderFulfilled for cab request done offline

// TODO:
// Do we need to buffer channels?
// Do we need to add ticker to distribute requests?
// Need to test if everyting on master side works


func create_cab_requests(floor int) []map[string]bool {
	cab_requests := make([]map[string]bool, floor)
	for f := 0; f < floor; f++ {
		cab_requests[f] = make(map[string]bool)
	}
	return cab_requests
}

func create_hall_assignments(floors, buttons int) [][]string {
	assignments := make([][]string, floors)
	for f := 0; f < floors; f++ {
		assignments[f] = make([]string, buttons)
	}
	return assignments
}

func new_master() *Master {
	return &Master{
		client_list : make(map[string]*Elevator_client),
		hall_requests: utilities.Create_request_arr(config.N_floors, config.N_buttons),
		hall_assignments: create_hall_assignments(config.N_floors, config.N_buttons),
		cab_requests:  create_cab_requests(config.N_floors),
	}
}

func (m *Master) Add_client(c *network.Client) {
	elev := &Elevator_client{
		Connection:    c,
		current_floor: 0,
		obstruction:   false,
		busy:          false,
		task_timer:    time.NewTimer(config.Request_timeout),
	}
	elev.task_timer.Stop()
	m.client_list[c.Addr] = elev
	log.Printf("Client connected: %s", c.Addr)
}

func (m *Master) Remove_client(addr string) {
	_, ok := m.client_list[addr]
	if ok {
		delete(m.client_list, addr)
		log.Printf("Client removed: %s", addr)
	}
}

func (m *Master) send_light_update() {
	for _, elev := range m.client_list {
		elev.Connection.Send(network.LightUpdate, &network.DataPayload{Lights: m.hall_requests})
	}
}

func (m *Master) find_closest_elevator(floor int) string {
	closest_addr := ""
	shortest_dist := 9999

	for _, client := range m.client_list {
		distance := utilities.Abs(client.current_floor - floor)
		if client.busy {
			distance += config.Busy_penalty
		}
		if distance < shortest_dist {
			closest_addr = client.Connection.Addr
			shortest_dist = distance
		}
	}
	return closest_addr
}

func (m *Master) distribute_request(floor int, button elevio.ButtonType, addr string) {
	if button == elevio.BT_Cab {
		m.cab_requests[floor][m.client_list[addr].ID] = true
		m.client_list[addr].busy = true
		m.client_list[addr].Connection.Send(network.OrderReceived, &network.DataPayload{OrderFloor: floor, OrderButton: button})
		m.client_list[addr].task_timer.Reset(config.Request_timeout)
	} else {
		if !m.hall_requests[floor][button] {
			client_addr := m.find_closest_elevator(floor)
			if client_addr != "" {
				m.hall_requests[floor][button] = true
				m.hall_assignments[floor][button] = m.client_list[addr].ID
				m.client_list[client_addr].busy = true
				m.client_list[client_addr].Connection.Send(network.OrderReceived, &network.DataPayload{OrderFloor: floor, OrderButton: button})
				m.send_light_update()
				m.client_list[addr].task_timer.Reset(config.Request_timeout)
			}
		}
	}
}

func (m *Master) redistribute_request(id string) {
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if m.hall_assignments[f][b] == id {
				m.distribute_request(f, b, "")
			}
		}
	}
}

func (m *Master) should_clear_busy(addr string) bool {
	for f := 0; f < config.N_floors; f++ {
		if m.cab_requests[f][m.client_list[addr].ID] {
			return false
		}	
	}
	for f := 0; f < config.N_floors; f++ {
		for b := elevio.ButtonType(0); b < config.N_buttons; b++ {
			if m.hall_assignments[f][b] == m.client_list[addr].ID {
				return false
			}
		}
	}
	return true
}

func (m *Master) client_timer_handler(timeout chan<- *network.Client) {
	for {
		for _, client := range m.client_list {
			if client.task_timer != nil {
				select {
				case <-client.task_timer.C:
					client.task_timer.Stop()
					timeout<- client.Connection

				default:
				
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func Run_master() {
	master := new_master()

	newChan := make(chan *network.Client)
	lossChan := make(chan *network.Client)
	msgChan := make(chan network.Message)
	

	go network.Start_server(lossChan, newChan, msgChan)
	go master.client_timer_handler(lossChan)

	for {
		select {
			case new := <-newChan:
				if len(master.client_list) == 0 {
					master.successor_addr = new.Addr
				}
				master.Add_client(new)

			case lost := <-lossChan:
				id := master.client_list[lost.Addr].ID
				master.Remove_client(lost.Addr)
				if len(master.client_list) == 0 {
					panic("Loss of internet")
				} else {
					if lost.Addr == master.successor_addr {
						for _, c := range master.client_list {
							master.successor_addr = c.Connection.Addr
							break
						}
					}
					master.redistribute_request(id)
				}

			case msg := <-msgChan:
				log.Printf("Recived message from %s", msg.Address)
				switch msg.Header {
				case network.OrderReceived:
					master.distribute_request(msg.Payload.OrderFloor, msg.Payload.OrderButton, msg.Address)

				case network.OrderFulfilled:
					if msg.Payload.OrderButton == elevio.BT_Cab {
						master.cab_requests[msg.Payload.OrderFloor][master.client_list[msg.Address].ID] = false
					} else {
						master.hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = false
						master.hall_assignments[msg.Payload.OrderFloor][msg.Payload.OrderButton] = ""
					}

					if master.should_clear_busy(msg.Address) {
						master.client_list[msg.Address].busy = false
						master.client_list[msg.Address].task_timer.Stop()
					}

				case network.FloorUpdate:
					master.client_list[msg.Address].current_floor = msg.Payload.CurrentFloor

				case network.ObstructionUpdate:
					master.client_list[msg.Address].obstruction = msg.Payload.Obstruction

				case network.ClientInfo:
					master.client_list[msg.Address].ID = msg.Payload.ID
					master.client_list[msg.Address].current_floor = msg.Payload.CurrentFloor
					master.client_list[msg.Address].obstruction = msg.Payload.Obstruction
				}
		}
	}
}