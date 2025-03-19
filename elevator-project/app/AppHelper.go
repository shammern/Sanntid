package app

import (
	"elevator-project/pkg/config"
	"elevator-project/pkg/drivers"
	"elevator-project/pkg/elevator"
	"strconv"
)

func convertOrderDataToOrders(orderData map[string][][2]bool) []elevator.Order {
	var ordersList []elevator.Order
	orders := orderData[strconv.Itoa(config.ElevatorID)]

	for floor, calls := range orders {
		if calls[0] { // Hall up call
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  true, // Default flag value, modify if needed
			})
		} else {
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallUp},
				Flag:  false, // Default flag value, modify if needed
			})
		}
		if calls[1] { // Hall down call
			ordersList = append(ordersList, elevator.Order{
				Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
				Flag:  true, // Default flag value, modify if needed
			})
		} else {
			ordersList = append(ordersList, elevator.Order{
			Event: drivers.ButtonEvent{Floor: floor, Button: drivers.BT_HallDown},
			Flag:  false, // Default flag value, modify if needed
		})
	}
	}
	
	return ordersList
}