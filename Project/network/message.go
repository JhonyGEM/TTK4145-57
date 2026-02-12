package network

import (
	"elevio"
)

type HeaderType int

const (
    OrderReceived HeaderType = iota
    OrderFulfilled
    FloorUpdate
    ObstructionUpdate
    ClientInfo
    Heartbeat
    LightUpdate
	Backup
	Ack
)

type DataPayload struct {
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
	Header  HeaderType      		`json:"header"`
	Payload *DataPayload			`json:"payload,omitempty"`
	Address string					`json:"address,omitempty"`
	UID     string                  `json:"uid,omitempty"`
}