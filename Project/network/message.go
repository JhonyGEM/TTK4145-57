package network

import (
	"project/elevio"
	"fmt"
)

type MessageType int

const (
    OrderReceived MessageType = iota
    OrderFulfilled
    FloorUpdate
    ObstructionUpdate
    ClientInfo
	Succesor
    Heartbeat
    LightUpdate
	Backup
	Ack
)

type MessagePayload struct {
	ID           string                 `json:"id,omitempty"`
	CurrentFloor int                    `json:"current_floor,omitempty"`
	Obstruction  bool					`json:"obstruction,omitempty"`
	OrderFloor   int                    `json:"order_floor,omitempty"`
	OrderButton  elevio.ButtonType      `json:"order_button,omitempty"`
	Lights       [][]bool               `json:"lights,omitempty"`
	BackupHall   [][]bool               `json:"backup_hall,omitempty"`
	BackupCab    []map[string]bool      `json:"backup_cab,omitempty"`
}

type Message struct {
	Header  MessageType      		`json:"header"`
	Payload *MessagePayload			`json:"payload,omitempty"`
	Address string					`json:"address,omitempty"`
	UID     string                  `json:"uid,omitempty"`
}

func (h MessageType) String() string {
	switch h {
	case OrderReceived:
		return "OrderReceived"
	case OrderFulfilled:
		return "OrderFulfilled"
	case FloorUpdate:
		return "FloorUpdate"
	case ObstructionUpdate:
		return "ObstructionUpdate"
	case ClientInfo:
		return "ClientInfo"
	case Succesor:
		return "Succesor"
	case Heartbeat:
		return "Heartbeat"
	case LightUpdate:
		return "LightUpdate"
	case Backup:
		return "Backup"
	case Ack:
		return "Ack"
	default:
		return fmt.Sprintf("HeaderType(%d)", int(h))
	}
}