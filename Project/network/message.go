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
	RequestCopy
)

type DataPayload struct {
	ID           string                 `json:"id,omitempty"`
	CurrentFloor int                    `json:"current_floor,omitempty"`
	Obstruction  bool					`json:"obstruction,omitempty"`
	OrderFloor   int                    `json:"order_floor,omitempty"`
	OrderButton  elevio.ButtonType      `json:"order_button,omitempty"`
	Lights       [][]bool               `json:"lights,omitempty"`
	CopyHallReq  [][]bool               `json:"copy_hall_req,omitempty"`
	CopyCabReq   []map[string]bool      `json:"copy_cab_req,omitempty"`
}

type Message struct {
	Header  HeaderType      		`json:"header"`
	Payload *DataPayload			`json:"payload,omitempty"`
	Address string					`json:"address,omitempty"`
}