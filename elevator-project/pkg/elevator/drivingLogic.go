package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/message"
	"fmt"
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

func (e *Elevator) clearHallReqsAtFloor() {
	switch e.travelDirection {
	case Up:
		if e.RequestMatrix.HallRequests[e.currentFloor][0] {
			e.RequestMatrix.HallRequests[e.currentFloor][0] = false
			completedOrderMsg := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallUp},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_HallUp))
			e.msgTx <- completedOrderMsg
		} else if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			completedOrderMsg := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown},
			}
			fmt.Printf("[ElevatorFSM] learing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_HallDown))
			e.msgTx <- completedOrderMsg

		}
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.RequestMatrix.CabRequests[e.currentFloor] = false
			completedOrderMsg := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_Cab},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_Cab))
			e.msgTx <- completedOrderMsg
		}
	case Down:
		if e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			completedOrderMsg := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_HallDown))
			e.msgTx <- completedOrderMsg
			//} else if !requestsBelow(rm, floor) {
			//	clear(HallUp)
		}
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			drivers.SetButtonLamp(drivers.BT_Cab, e.currentFloor, false)
			e.RequestMatrix.CabRequests[e.currentFloor] = false
			completedOrderMsg := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_Cab},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_Cab))
			e.msgTx <- completedOrderMsg
		}
	case Stop:
		if e.RequestMatrix.CabRequests[e.currentFloor] {
			e.RequestMatrix.CabRequests[e.currentFloor] = false
		}

		if e.RequestMatrix.HallRequests[e.currentFloor][0] || e.RequestMatrix.HallRequests[e.currentFloor][1] {
			e.RequestMatrix.HallRequests[e.currentFloor][0] = false
			completedOrderMsg1 := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallUp},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_HallUp))
			e.msgTx <- completedOrderMsg1

			//Add sleep?
			e.RequestMatrix.HallRequests[e.currentFloor][1] = false
			completedOrderMsg2 := message.Message{
				Type:        message.CompletedOrder,
				ElevatorID:  e.ElevatorID,
				MsgID:       e.counter.Next(),
				ButtonEvent: drivers.ButtonEvent{Floor: e.currentFloor, Button: drivers.BT_HallDown},
			}
			fmt.Printf("[ElevatorFSM] Clearing order: Floor: %d, Type: %d\n", e.currentFloor, int(drivers.BT_HallDown))
			e.msgTx <- completedOrderMsg2
		}
	}
}
