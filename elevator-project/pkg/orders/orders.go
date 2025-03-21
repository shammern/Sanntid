package orders

import (
	"elevator-project/pkg/drivers"
	"errors"
	"fmt"
)

type RequestMatrix struct {
	HallRequests [][2]bool
	CabRequests  []bool
}

func NewRequestMatrix(numFloors int) *RequestMatrix {
	rm := &RequestMatrix{
		HallRequests: make([][2]bool, numFloors),
		CabRequests:  make([]bool, numFloors),
	}
	return rm
}

func (rm *RequestMatrix) SetHallRequest(floor int, direction int, active bool) error {
	if floor < 0 || floor >= len(rm.HallRequests) {
		return errors.New("floor out of range")
	}
	if direction < 0 || direction > 1 {
		return errors.New("invalid hall direction")
	}
	rm.HallRequests[floor][direction] = active
	return nil
}

func (rm *RequestMatrix) SetCabRequest(floor int, active bool) error {
	if floor < 0 || floor >= len(rm.CabRequests) {
		return errors.New("floor out of range")
	}
	rm.CabRequests[floor] = active
	return nil
}

func (rm *RequestMatrix) ClearHallRequest(floor int, direction int) error {
	return rm.SetHallRequest(floor, direction, false)
}

func (rm *RequestMatrix) ClearCabRequest(floor int) error {
	return rm.SetCabRequest(floor, false)
}

func (rm *RequestMatrix) HasHallRequest(floor int, direction int) (bool, error) {
	if floor < 0 || floor >= len(rm.HallRequests) {
		return false, errors.New("floor out of range")
	}
	if direction < 0 || direction > 1 {
		return false, errors.New("invalid hall direction")
	}
	return rm.HallRequests[floor][direction], nil
}

func (rm *RequestMatrix) HasCabRequest(floor int) (bool, error) {
	if floor < 0 || floor >= len(rm.CabRequests) {
		return false, errors.New("floor out of range")
	}
	return rm.CabRequests[floor], nil
}

func (rm *RequestMatrix) DebugPrint() {
	fmt.Println("==== Request Matrix ====")
	for floor, hall := range rm.HallRequests {
		cabStr := "off"
		if rm.CabRequests[floor] {
			cabStr = "on"
		}
		hallUpStr := "off"
		if hall[0] {
			hallUpStr = "on"
		}
		hallDownStr := "off"
		if hall[1] {
			hallDownStr = "on"
		}
		fmt.Printf("Floor %d: Cab: %s, Hall Up: %s, Hall Down: %s\n", floor, cabStr, hallUpStr, hallDownStr)
	}
	fmt.Println("========================")
}

func GetUnassignedOrders(rm *RequestMatrix) []drivers.ButtonEvent {
	var orders []drivers.ButtonEvent

	fmt.Println("[DEBUG] Checking request matrix...")

	// Ensure RequestMatrix is properly initialized
	if rm == nil {
		fmt.Println("[ERROR] RequestMatrix is nil!")
		return orders
	}

	for floor, hallReq := range rm.HallRequests {
		fmt.Printf("[DEBUG] Floor %d HallRequests: %+v\n", floor, hallReq)
		for dir, active := range hallReq {
			if active {
				fmt.Printf("[DEBUG] Found active request at Floor %d Direction %d\n", floor, dir)
				orders = append(orders, drivers.ButtonEvent{
					Floor:  floor,
					Button: drivers.ButtonType(dir),
				})
			}
		}
	}
	return orders
}
