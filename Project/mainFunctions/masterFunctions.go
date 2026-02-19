package mainfunctions

import (
	"project/master"
	"project/network"
)

func AddElevatorHandler(master *master.Master, newElevator *network.Client) {
	master.Add_client(newElevator)
	if len(master.Client_list) == 1 {
		master.Successor_addr = newElevator.Addr
		master.Client_list[newElevator.Addr].Send(
			network.Message{
				Header: network.Successor,
			})
	}
}

func RemoveClient(m *master.Master, conn *network.Client) {
	id := m.Client_list[conn.Addr].ID
	m.Remove_client(conn.Addr)

	if len(m.Client_list) == 0 {
		panic("Loss of connectivity")
	} else {
		if conn.Addr == m.Successor_addr {
			for _, c := range m.Client_list {
				m.Successor_addr = c.Connection.Addr
				break
			}
		}

		m.Redistribute_request(id)
	}
}

func Resend(mast *master.Master) {
	if len(mast.Client_list) > 0 {
		mast.Resend_hall_request()
	}
}
