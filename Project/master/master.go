package master
import(
	"network"
	"log"
)

type Master struct {
	clients map[*network.Client]struct{}
}

func new_master() *Master {
	return &Master{
		clients : make(map[*network.Client]struct{}),
	}
}

func (m *Master) Add_client(c *network.Client) {
	m.clients[c] = struct{}{}
	log.Printf("Client connected: %s", c.Addr)
}

func (m *Master) Remove_client(c *network.Client) {
	_, ok := m.clients[c]
	if ok {
		delete(m.clients, c)
		log.Printf("Client removed: %s", c.Addr)
	}
}

func Run_master() {
	master := new_master()

	newChan := make(chan *network.Client)
	lossChan := make(chan *network.Client)
	msgChan := make(chan network.Message)

	go network.Start_server(lossChan, newChan, msgChan)

	for {
		select {
			case new := <-newChan:
				master.Add_client(new)

			case lost := <-lossChan:
				master.Remove_client(lost)

			case msg := <-msgChan:
				log.Printf("Recived: %s", msg.Header)
				switch msg.Header {
				}
		}
	}
}