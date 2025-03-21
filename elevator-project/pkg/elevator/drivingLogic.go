package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
)

func (e *Elevator) requestsAbove() bool {
	for i := e.currentFloor + 1; i < len(e.RequestMatrix.CabRequests); i++ {
		if e.RequestMatrix.CabRequests[i] || e.RequestMatrix.HallRequests[i][0] || e.RequestMatrix.HallRequests[i][1] {
			return true
		}
	}
	return false
}

func (e *Elevator) requestsBelow() bool {
	for i := 0; i < e.currentFloor; i++ {
		if e.RequestMatrix.CabRequests[i] || e.RequestMatrix.HallRequests[i][0] || e.RequestMatrix.HallRequests[i][1] {
			return true
		}
	}
	return false
}

// anyRequestsAtFloor returns true if any request exists at the specified floor.
func (e *Elevator) anyRequestsAtFloor() bool {
	if e.currentFloor < 0 || e.currentFloor >= len(e.RequestMatrix.CabRequests) {
		return false
	}
	return e.RequestMatrix.CabRequests[e.currentFloor] || e.RequestMatrix.HallRequests[e.currentFloor][0] || e.RequestMatrix.HallRequests[e.currentFloor][1]
}

// shouldStop returns true if the elevator should stop at the given floor,
// based on the current direction and requests.
func (e *Elevator) shouldStop() bool {
	// Boundary check.
	if e.currentFloor < 0 || e.currentFloor >= len(e.RequestMatrix.CabRequests) {
		return false
	}
	switch e.travelDirection {
	case Up:
		return e.RequestMatrix.HallRequests[e.currentFloor][0] ||
			e.RequestMatrix.CabRequests[e.currentFloor] ||
			!e.requestsAbove() ||
			e.currentFloor == 0 ||
			e.currentFloor == len(e.RequestMatrix.CabRequests)-1
	case Down:
		return e.RequestMatrix.HallRequests[e.currentFloor][1] ||
			e.RequestMatrix.CabRequests[e.currentFloor] ||
			!e.requestsBelow() ||
			e.currentFloor == 0 ||
			e.currentFloor == len(e.RequestMatrix.CabRequests)-1
	case Stop:
		return true
	default:
		return false
	}
}

// Downward bias
func (e *Elevator) chooseDirection() Direction {
	switch e.travelDirection {
	case Up:
		if e.requestsAbove() {
			return Up

		} else if e.anyRequestsAtFloor() {
			return Stop

		} else if e.requestsBelow() {
			return Down

		} else {
			return Stop
		}

	case Down, Stop:
		if e.requestsBelow() {
			return Down

		} else if e.anyRequestsAtFloor() {
			return Stop

		} else if e.requestsAbove() {
			return Up

		} else {
			return Stop
		}

	default:
		return Stop
	}
}

// TODO: This is a big switchcase function, can it be written better?
// Checks if there is an active request at the floor that should be cleared. Sends a completedordermsg on the network if clearing an order
func (e *Elevator) clearHallReqsAtFloor() {
	switch e.travelDirection {
	case Up:
		if e.RequestMatrix.HallRequests[e.currentFloor][0] {
			e.RequestMatrix.HallRequests[e.currentFloor][0] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallUp})
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
		} else if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)

		}
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.RequestMatrix.CabRequests[e.currentFloor] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_Cab})
		}
	case Down:
		if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
		}
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.RequestMatrix.CabRequests[e.currentFloor] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_Cab})
		}
	case Stop:
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			e.RequestMatrix.CabRequests[e.currentFloor] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_Cab})
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
		}

		if e.RequestMatrix.HallRequests[e.currentFloor][0] {
			e.RequestMatrix.HallRequests[e.currentFloor][0] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallUp})
			drivers.SetButtonLamp(drivers.BT_HallUp, e.currentFloor, false)
		} else if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			e.NotifyMaster(message.CompletedOrder, drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown})
			drivers.SetButtonLamp(drivers.BT_HallDown, e.currentFloor, false)
		}
	}
}
