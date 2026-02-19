package network

import (
	"fmt"
	"project/elevio"
)

type HeaderType int

const (
	OrderReceived HeaderType = iota
	OrderFulfilled
	FloorUpdate
	ObstructionUpdate
	ClientInfo
	Successor
	Heartbeat
	LightUpdate
	Backup
	Ack
)

type DataPayload struct {
	ID           string            `json:"id,omitempty"`
	CurrentFloor int               `json:"current_floor,omitempty"`
	Obstruction  bool              `json:"obstruction,omitempty"`
	OrderFloor   int               `json:"order_floor,omitempty"`
	OrderButton  elevio.ButtonType `json:"order_button,omitempty"`
	Lights       [][]bool          `json:"lights,omitempty"`
	BackupHall   [][]bool          `json:"backup_hall,omitempty"`
	BackupCab    []map[string]bool `json:"backup_cab,omitempty"`
}

type Message struct {
	Header  HeaderType   `json:"header"`
	Payload *DataPayload `json:"payload,omitempty"`
	Address string       `json:"address,omitempty"`
	UID     string       `json:"uid,omitempty"`
}

func (h HeaderType) String() string {
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
	case Successor:
		return "Successor"
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
