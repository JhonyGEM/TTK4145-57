package mainfunctions

import (
	"log"
	"project/config"
	"project/elevator"
	"project/elevio"
	"project/master"
	"project/network"
)

type Backup struct {
	BackupHallReg [][]bool
	BackupCabReg  []map[string]bool
}

func ElevatorOrderReceivedMessage(elevator *elevator.Elevator, message network.Message) {
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

func ElevatorACKHandler(elevator *elevator.Elevator, message network.Message) {
	_, ok := elevator.Pending[message.UID]
	if ok {
		delete(elevator.Pending, message.UID)
		elevator.Save_pending()
	}
}

func MasterOrderReceivedMessage(mast *master.Master, msg network.Message) {
	if msg.Payload.OrderButton == elevio.BT_Cab {
		mast.Cab_requests[msg.Payload.OrderFloor][mast.Client_list[msg.Address].ID] = true
	} else {
		mast.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = true
	}
	mast.Client_list[mast.Successor_addr].Send(network.Message{Header: network.Backup, Payload: &network.DataPayload{BackupHall: mast.Hall_requests, BackupCab: mast.Cab_requests}, UID: msg.UID})
	mast.Pending[msg.UID] = &msg
}

func OrderFulfilledMessage(mast *master.Master, msg network.Message) {
	if msg.Payload.OrderButton == elevio.BT_Cab {
		mast.Cab_requests[msg.Payload.OrderFloor][mast.Client_list[msg.Address].ID] = false
	} else {
		mast.Hall_requests[msg.Payload.OrderFloor][msg.Payload.OrderButton] = false
		mast.Hall_assignments[msg.Payload.OrderFloor][msg.Payload.OrderButton] = ""
	}
	mast.Send_light_update()

	if mast.Still_busy(msg.Address) {
		mast.Client_list[msg.Address].Busy = false
		mast.Client_list[msg.Address].Task_timer.Stop()
	} else {
		mast.Client_list[msg.Address].Task_timer.Reset(config.Request_timeout)
	}
}

func MasterACKHandler(mast *master.Master, msg network.Message) {
	message := mast.Pending[msg.UID]

	_, ok := mast.Pending[msg.UID]
	if ok {
		mast.Distribute_request(message.Payload.OrderFloor, message.Payload.OrderButton, message.Address)
		mast.Client_list[message.Address].Send(network.Message{Header: network.Ack, UID: message.UID})
		delete(mast.Pending, msg.UID)
	}
}

func ClientInfoMessage(mast *master.Master, msg network.Message) {
	mast.Client_list[msg.Address].ID = msg.Payload.ID
	mast.Client_list[msg.Address].Current_floor = msg.Payload.CurrentFloor
	mast.Client_list[msg.Address].Obstruction = msg.Payload.Obstruction
	mast.Resend_cab_request(msg.Address)
}
