package elevator

import (
	"project/config"
	"project/elevio"
	"project/network"
	"project/utilities"
	"time"
)

func (ls *LocalState) RequestPending() bool {
	for floor := 0; floor < config.N_floors; floor++ {
		for button := elevio.ButtonType(0); button < config.N_buttons; button++ {
			if ls.Requests[floor][button] {
				return true
			}
		}
	}
	return false
}

func (ls *LocalState) requestAbove() bool {
	for floor := ls.CurrentFloor + 1; floor < config.N_floors; floor++ {
		for button := elevio.ButtonType(0); button < config.N_buttons; button++ {
			if ls.Requests[floor][button] {
				return true
			}
		}
	}
	return false
}

func (ls *LocalState) requestBelow() bool {
	for floor := 0; floor < ls.CurrentFloor; floor++ {
		for button := elevio.ButtonType(0); button < config.N_buttons; button++ {
			if ls.Requests[floor][button] {
				return true
			}
		}
	}
	return false
}

func (ls *LocalState) requestHere() bool {
	for button := elevio.ButtonType(0); button < config.N_buttons; button++ {
		if ls.Requests[ls.CurrentFloor][button] {
			return true
		}
	}
	return false
}

func (ls *LocalState) clearHallRequests() {
	for floor := 0; floor < config.N_floors; floor++ {
		ls.Requests[floor][elevio.BT_HallUp] = false
		ls.Requests[floor][elevio.BT_HallDown] = false
	}
}

func (ls *LocalState) shouldStop() bool {
	switch ls.Direction {
	case elevio.MD_Down:
		return ls.Requests[ls.CurrentFloor][elevio.BT_HallDown] || ls.Requests[ls.CurrentFloor][elevio.BT_Cab] || !ls.requestBelow()

	case elevio.MD_Up:
		return ls.Requests[ls.CurrentFloor][elevio.BT_HallUp] || ls.Requests[ls.CurrentFloor][elevio.BT_Cab] || !ls.requestAbove()

	case elevio.MD_Stop:
		return !ls.requestBelow() && !ls.requestAbove()
	
	default:
		return false
	}
}

func (ls *LocalState) shouldClear() bool {
	return ((ls.Direction == elevio.MD_Up && ls.Requests[ls.CurrentFloor][elevio.BT_HallUp]) ||
		    (ls.Direction == elevio.MD_Down && ls.Requests[ls.CurrentFloor][elevio.BT_HallDown]) ||
		    ls.Direction == elevio.MD_Stop ||
		    ls.Requests[ls.CurrentFloor][elevio.BT_Cab])
}

func (e *Elevator) clearRequest(floor int, button elevio.ButtonType) {
	if !e.Local.Requests[floor][button] {
		return
	}
	e.Local.Requests[floor][button] = false
	if e.Network.IsConnected {
		e.Network.Send(network.Message{
			Header: network.OrderFulfilled, 
			Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: button}})
	}
}

func (e *Elevator) clearAtCurrentFloor() {
	e.clearRequest(e.Local.CurrentFloor, elevio.BT_Cab)

	if (e.Local.requestAbove() && (e.Local.Requests[e.Local.CurrentFloor][elevio.BT_HallUp] && e.Local.Requests[e.Local.CurrentFloor][elevio.BT_HallDown])) {
		e.clearRequest(e.Local.CurrentFloor, elevio.BT_HallUp)
	} else if (e.Local.requestBelow() && (e.Local.Requests[e.Local.CurrentFloor][elevio.BT_HallUp] && e.Local.Requests[e.Local.CurrentFloor][elevio.BT_HallDown])) {
		e.clearRequest(e.Local.CurrentFloor, elevio.BT_HallDown)
	} else {
		e.clearRequest(e.Local.CurrentFloor, elevio.BT_HallUp)
		e.clearRequest(e.Local.CurrentFloor, elevio.BT_HallDown)
	}
}

func (ns *NetworkState) ResendPendingRequest() {
	if ns.IsConnected {
		for uid, pendingMsg := range ns.Pending {
			if time.Since(pendingMsg.Timestamp) > config.Pending_timeout {
				ns.Send(pendingMsg.Message)
				ns.Pending[uid].Timestamp = time.Now()
				utilities.SaveToFile(config.Pending_backup, ns.Pending)
			}
		}
	}
}

// Saves active cab requests on master promotion
func (e *Elevator) SaveCabRequests() {
	cabRequest := make(map[string]*network.Pending)
	for floor := 0; floor < config.N_floors; floor++ {
		if e.Local.Requests[floor][elevio.BT_Cab] {
			uid := utilities.GenUID(e.Network.ID, e.Network.Sequence)
			e.Network.Sequence++
			cabRequest[uid] = &network.Pending{Message: network.Message{
				Header: network.OrderReceived, 
				Payload: &network.MessagePayload{OrderFloor: floor, OrderButton: elevio.BT_Cab}}}
		}
	}
	utilities.SaveToFile(config.Elev_backup, cabRequest)
}

func (ls *LocalState) LoadCabRequests() {
	cabRequest := utilities.LoadFromFile(config.Elev_backup)

	for uid, request := range cabRequest {
		if request.Message.Header == network.OrderReceived {
			ls.Requests[request.Message.Payload.OrderFloor][request.Message.Payload.OrderButton] = true
			delete(cabRequest, uid)
		}
	}
	ls.SetButtonLights()
	utilities.SaveToFile(config.Elev_backup, cabRequest)
}
