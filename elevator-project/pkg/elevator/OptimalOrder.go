package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
)

//This function is for single elevator setup, multiple elevator handler is not yet fully implemented
func OptimalAssignment(rm *orders.RequestMatrix, currentFloor int, currentDirection drivers.MotorDirection) (drivers.ButtonEvent, bool) {
	var bestOrder drivers.ButtonEvent
	bestCost := int(^uint(0) >> 1)
	found := false

	for floor, active := range rm.CabRequests {
		if active {
			cost := computeCost(floor, drivers.BT_Cab, currentFloor, currentDirection)
			if cost < bestCost {
				bestCost = cost
				bestOrder = drivers.ButtonEvent{Floor: floor, Button: drivers.BT_Cab}
				found = true
			}
		}
	}

	for floor, reqs := range rm.HallRequests {
		for dir, active := range reqs {
			if active {
				var button drivers.ButtonType
				if dir == 0 {
					button = drivers.BT_HallUp
				} else {
					button = drivers.BT_HallDown
				}
				cost := computeCost(floor, button, currentFloor, currentDirection)
				if cost < bestCost {
					bestCost = cost
					bestOrder = drivers.ButtonEvent{Floor: floor, Button: button}
					found = true
				}
			}
		}
	}

	return bestOrder, found
}


func computeCost(requestFloor int, requestButton drivers.ButtonType, currentFloor int, currentDirection drivers.MotorDirection) int {
	baseCost := abs(currentFloor - requestFloor)
	penalty := 1000
	switch currentDirection {
	case drivers.MD_Up:
		if requestFloor >= currentFloor && (requestButton == drivers.BT_HallUp || requestButton == drivers.BT_Cab) {
			return baseCost
		}
		return baseCost + penalty
	case drivers.MD_Down:
		if requestFloor <= currentFloor && (requestButton == drivers.BT_HallDown || requestButton == drivers.BT_Cab) {
			return baseCost
		}
		return baseCost + penalty
	default: 
		return baseCost
	}
}
