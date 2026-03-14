package network

import (
	"project/elevio"
	"fmt"
	"time"
)

type MessageType int

const (
    OrderReceived MessageType = iota
    OrderFulfilled
    FloorUpdate
    ObstructionUpdate
    ClientInfo
	Successor
	NotSuccessor
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

type Pending struct {
	Timestamp time.Time				`json:"timestamp,omitempty"`
	Message   Message				`json:"message"`
}

func (m MessageType) String() string {
	switch m {
	case OrderReceived:
		return "Order received"
	case OrderFulfilled:
		return "Order fulfilled"
	case FloorUpdate:
		return "Floor update"
	case ObstructionUpdate:
		return "Obstruction update"
	case ClientInfo:
		return "Client info"
	case Successor:
		return "Successor"
	case NotSuccessor:
		return "Not successor"
	case Heartbeat:
		return "Heartbeat"
	case LightUpdate:
		return "Light update"
	case Backup:
		return "Backup"
	case Ack:
		return "Ack"
	default:
		return fmt.Sprintf("HeaderType(%d)", int(m))
	}
}