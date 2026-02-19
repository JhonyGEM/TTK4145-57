package mainFunctions

import (
	"log"
	"project/elevator"
	"project/network"
)

type Backup struct {
	BackupHallReg [][]bool
	BackupCabReg  []map[string]bool
}

func OrderReceivedMessage(elevator *elevator.Elevator, message network.Message) {
	elevator.Requests[message.Payload.OrderFloor][message.Payload.OrderButton] = true
	elevator.Save_pending()
	elevator.Step_FSM()
}

func BackupMessage(back *Backup, elevator *elevator.Elevator, message network.Message) {
	log.Printf("Recieved backup")

	back.BackupCabReg = message.Payload.BackupCab
	back.BackupHallReg = message.Payload.BackupHall

	elevator.Connection.Send(network.Message{
		Header: network.Ack,
		UID:    message.UID,
	})
}

func ACKHandler(elevator *elevator.Elevator, message network.Message) {
	_, ok := elevator.Pending[message.UID]
	if ok {
		delete(elevator.Pending, message.UID)
		elevator.Save_pending()
	}
}
