package elevator

import (
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/orders"
	"math"
)

//Based on single elevator setup, not tuned to multiple elevator setup.
func DynamicQueueManager(newOrders <-chan drivers.ButtonEvent, elevatorReady <-chan int, sendOrder chan<- drivers.ButtonEvent, numFloors int) {
	rm := orders.NewRequestMatrix(numFloors)
	for {
		select {
		case order := <-newOrders:
			switch order.Button {
			case drivers.BT_Cab:
				_ = rm.SetCabRequest(order.Floor, true)
			case drivers.BT_HallUp:
				_ = rm.SetHallRequest(order.Floor, 0, true)
			case drivers.BT_HallDown:
				_ = rm.SetHallRequest(order.Floor, 1, true)
			}
			drivers.SetButtonLamp(order.Button, order.Floor, true)

		case currentFloor := <-elevatorReady:
			if nextOrder, found := chooseNextOrder(rm, currentFloor); found {
				sendOrder <- nextOrder
			}
		}
	}
}

func chooseNextOrder(rm *orders.RequestMatrix, currentFloor int) (drivers.ButtonEvent, bool) {
	bestFloor := -1
	var bestButton drivers.ButtonType = drivers.BT_Cab
	minDistance := math.MaxInt32
	found := false

	for floor, active := range rm.CabRequests {
		if active {
			dist := abs(currentFloor - floor)
			if dist < minDistance {
				minDistance = dist
				bestFloor = floor
				bestButton = drivers.BT_Cab
				found = true
			}
		}
	}
	for floor, reqs := range rm.HallRequests {
		for dir, active := range reqs {
			if active {
				dist := abs(currentFloor - floor)
				if dist < minDistance {
					minDistance = dist
					bestFloor = floor
					if dir == 0 {
						bestButton = drivers.BT_HallUp
					} else {
						bestButton = drivers.BT_HallDown
					}
					found = true
				}
			}
		}
	}
	if found {
		return drivers.ButtonEvent{Floor: bestFloor, Button: bestButton}, true
	}
	return drivers.ButtonEvent{}, false
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
